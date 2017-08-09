package main

import (
    "fmt"
    "log"
    "time"
    "strconv"
    "strings"
    "encoding/json"
    //"github.com/davecgh/go-spew/spew"
    "github.com/influxdata/influxdb/client/v2"
)

type Influx struct {
    client     client.Client          // influx api client
    notifer    *Notifer               // notifer to send alters to

    options    map[string]string      // TODO: replace with explicit declarations (remove parsing actions from Influx)
    templates  map[string]*Template
    checks     []*Check
}

type Template struct {
    Name   string
    body   string
    args   []string
}

type Check struct {
    name     string
    values   []string
    trigger  *Trigger
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
    })
    if err != nil {
        log.Fatalln(err)
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

func (i *Influx) RunWith(notifer *Notifer) {
    interval, err := strconv.Atoi(i.options["interval"])

    if err != nil {
        log.Fatalf("[e] influx: wrong 'interval' value '%s'\n", i.options["interval"])
    }

    i.notifer = notifer
    i.initTriggers(interval)

    for {
        log.Println("[i] influx: check loop...")

        for _, check := range i.checks {
            log.Println("[i] influx: checking", check.name)

            sql := i.getSqlForCheck(check)
            log.Println("[i] influx: sql =", strings.TrimSpace(sql))

            value, err := i.query(sql)
            if err != nil {
                log.Fatalln("[w] influx: error during query execution;", err)
            }

            log.Println("[i] influx: value =", value)

            if value == nil {
                log.Println("[i] influx: touching trigger with '0' value instead of 'nil'")
                check.trigger.Touch(0)
                continue
            }

            check.trigger.Touch(value)

            log.Println("")

        }
        time.Sleep(time.Second * time.Duration(interval))
    }
}

func (i *Influx) initTriggers(interval int) {
    channel := i.options["alert"]

    for _, check := range i.checks {
        _name := check.name  // catch var for closure
        _check := check      // catch var for closure

        check.trigger.callback = func(state *State, lastValue interface{}) error {
            var err error

            details := i.getDetailsForCheck(_check)
            details["value"] = fmt.Sprintf("%v", lastValue)

            sql := i.getSqlForCheck(_check)

            switch state.Name {
                case "good": err = i.notifer.Good(
                    channel,
                    fmt.Sprintf("QUERY: %s", _name),
                    fmt.Sprintf("Query \"%s\" is good more than %d sec. ```%s```", _name, interval * state.Cycles, sql),
                    details,
                )
                case "warn": err = i.notifer.Warn(
                    channel,
                    fmt.Sprintf("QUERY: %s", _name),
                    fmt.Sprintf("*WARN:* query \"%s\" has bad value for more than %d sec. ```%s```", _name, interval * state.Cycles, sql),
                    details,
                )
                case "crit": err = i.notifer.Crit(
                    channel,
                    fmt.Sprintf("QUERY: %s", _name),
                    fmt.Sprintf("*CRITICAL:* query \"%s\" has bad value for more than %d sec. ```%s```", _name, interval * state.Cycles, sql),
                    details,
                )
            }
            return err
        }
    }
}

func (i *Influx) getDetailsForCheck(check *Check) map[string]string {
    tpl, _ := i.templates[check.name]
    str := ""
    for i, arg := range tpl.args {
        str += fmt.Sprintf("%s = \"%s\" \n", arg, check.values[i])
    }

    return map[string]string{
        "args": str,
    }
}

func (i *Influx) getSqlForCheck(check *Check) string {
    tpl, ok := i.templates[check.name]
    if !ok {
        log.Fatalf("[e] influx: missing template '%s'\n", check.name)
    }

    return tpl.Format(check.values...)
}

func (t *Template) Format(values ...string) string {
    if len(values) != len(t.args) {
        log.Fatalln("wrong call of template", t.Name, " - wrong amount of arguments")
    }

    res := t.body
    for i, arg := range t.args {
        res = strings.Replace(res, "%" + arg, values[i], -1)
    }

    return strings.TrimSpace(res)
}
