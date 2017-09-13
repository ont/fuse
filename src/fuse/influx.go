package main

import (
    "io"
    "fmt"
    "time"
    "strconv"
    "strings"
    "crypto/md5"
    "encoding/json"
    //"github.com/davecgh/go-spew/spew"
    "github.com/influxdata/influxdb/client/v2"
    log "github.com/sirupsen/logrus"
)

type Influx struct {
    client     client.Client          // influx api client
    notifer    *Notifer               // notifer to send alters to

    options    map[string]string      // TODO: replace with explicit declarations (remove parsing actions from Influx)
    templates  map[string]*Template
    checks     []*Check
}

type Template struct {
    Name     string
    body     string
    preview  string
    args     []string
}

type Check struct {
    template  string     // name of template
    info      string     // info string for alert message
    values    []string
    trigger   *Trigger
}

// TODO: add constructor and save calculation into cache-field
func (c *Check) GetReportId() string {
    h := md5.New()
    io.WriteString(h, c.template + "|" + c.info + "|" + strings.Join(c.values,"|"))
    return fmt.Sprintf("%.5x", h.Sum(nil))
}

func NewInflux(options map[string]string) *Influx {
    optionsFull := map[string]string{
        "url" : "localhost:8086",
        "database" : "telegraf",
        "interval" : "5",
        "alert"    : "",
    }
    for k,v := range options {
        optionsFull[k] = v
    }

    c, err := client.NewHTTPClient(client.HTTPConfig{
        Addr: optionsFull["url"],
        Timeout: 15 * time.Second,
    })
    if err != nil {
        log.Fatalln("influx: ", err)
    }

    return &Influx{
        client: c,
        options: optionsFull,
        templates: make(map[string]*Template),
        checks: make([]*Check, 0),
    }
}

func (i *Influx) AddTemplate(template *Template) {
    i.templates[template.Name] = template
}

func (i *Influx) AddCheck(check *Check) {
    i.checks = append(i.checks, check)
}

func (i *Influx) query(cmd string) (interface{}, error) {
    // Create a new HTTPClient
    q := client.Query{
        Command: cmd,         //"select count(http_size) from \"30days\".http where http_path =~ /api\\/.*\\/tasks\\/show/ and http_code =~ /2../ and time > now() - 3h;",
        Database: i.options["database"], //"telegraf",
    }

    res, err := i.client.Query(q)
    if err != nil {
        return 0, err
    }

    if res.Error() != nil {
        return 0, res.Error()
    }

    if res.Results[0].Series == nil {
        return nil, nil
    }

    value := res.Results[0].Series[0].Values[0][1]

    if number, ok := value.(json.Number); ok {
        return number.Float64()
    }

    if str, ok := value.(string); ok {
        return str, nil
    }

    return nil, fmt.Errorf("result is not nor json.Number nor string")
}

func (i *Influx) GetName() string {
    return "influx"
}

func (i *Influx) RunWith(notifer *Notifer) {
    interval, err := strconv.Atoi(i.options["interval"])

    if err != nil {
        log.WithFields(log.Fields{"value" : i.options["interval"]}).Fatal("influx: wrong 'interval' value")
    }

    i.notifer = notifer
    i.initTriggers(interval)

    for {
        log.Info("influx: check loop...")

        for _, check := range i.checks {

            log.WithFields(log.Fields{"info" : check.info}).Debug("influx: next check")

            sql := i.getSqlForCheck(check)
            log.WithFields(log.Fields{"sql" : strings.TrimSpace(sql)}).Debug("influx: executing sql")

            value, err := i.query(sql)
            if err != nil {
                log.Error("influx: error during query execution: ", err)
                continue
            }

            log.WithFields(log.Fields{"value" : value}).Debug("influx: sending value to trigger")

            if value == nil {
                log.Debug("influx: touching trigger with '0' value instead of 'nil'")
                check.trigger.Touch(0)
                continue
            }

            check.trigger.Touch(value)
        }

        time.Sleep(time.Duration(interval) * time.Second)
    }
}

func (i *Influx) initTriggers(interval int) {
    channel := i.options["alert"]

    for _, check := range i.checks {
        _check := check  // catch var for closure

        // assign new callback-closure
        check.trigger.callback = func(state *State, lastValue interface{}) error {
            details := i.getDetailsForCheck(_check)
            details["value"] = fmt.Sprintf("%v", lastValue)
            details["template"] = _check.template
            details["report-id"] = _check.GetReportId()

            sql := i.getSqlForCheck(_check)

            var body string
            switch state.Name {
                case "good":
                    body = fmt.Sprintf("Query is good more than %d sec. ```%s```", interval * state.Cycles, sql)
                case "warn":
                    body = fmt.Sprintf("*WARN:* query has bad value for more than %d sec. ```%s```", interval * state.Cycles, sql)
                case "crit":
                    body = fmt.Sprintf("*CRITICAL:* query has bad value for more than %d sec. ```%s```", interval * state.Cycles, sql)
            }

            err := i.notifer.Notify(
                state.Name,  // notify level
                channel,
                Message{
                    IconUrl: "https://aperogeek.fr/wp-content/uploads/2017/04/influx_logo.png",  // TODO: replace
                    From: "influx",
                    Title: fmt.Sprintf("QUERY: %s", _check.info),
                    Body: body,
                    Details: details,
                },
            )
            return err
        }
    }
}

func (i *Influx) getDetailsForCheck(check *Check) map[string]string {
    tpl, _ := i.templates[check.template]
    str := ""
    for i, arg := range tpl.args {
        str += fmt.Sprintf("%s = \"%s\" \n", arg, check.values[i])
    }

    return map[string]string{
        "args": str,
    }
}

func (i *Influx) getSqlForCheck(check *Check) string {
    tpl, ok := i.templates[check.template]
    if !ok {
        log.WithFields(log.Fields{ "module" : "influx" }).Fatalf("influx: missing template '%s'\n", check.template)
    }

    return tpl.Format(check.values...)
}

func (t *Template) Format(values ...string) string {
    return t.formatTemplate(t.body, values)
}

func (t *Template) FormatPreview(values ...string) string {
    return t.formatTemplate(t.preview, values)
}

func (t *Template) formatTemplate(tpl string, values []string) string {
    if len(values) != len(t.args) {
        log.WithFields(log.Fields{ "values" : values, "args" : t.args }).Fatalln("influx: wrong call of template", t.Name, " - wrong amount of arguments")
    }

    res := tpl
    for i, arg := range t.args {
        res = strings.Replace(res, "%" + arg, values[i], -1)
    }

    return strings.TrimSpace(res)
}
