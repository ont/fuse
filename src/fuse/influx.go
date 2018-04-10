package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	//"github.com/davecgh/go-spew/spew"
	"github.com/influxdata/influxdb/client/v2"
	log "github.com/sirupsen/logrus"
)

type Influx struct {
	client  client.Client // influx api client
	notifer *Notifer      // notifer to send alters to

	options   map[string]string // TODO: replace with explicit declarations (remove parsing actions from Influx)
	templates map[string]*Template
	checks    []*Check
}

type Template struct {
	Name    string
	body    string
	preview string
	args    []string
}

type Check struct {
	template string // name of template
	info     string // info string for alert message
	values   []string
	trigger  *Trigger
}

// TODO: add constructor and save calculation into cache-field
func (c *Check) GetReportId() string {
	h := md5.New()
	io.WriteString(h, c.template+"|"+c.info+"|"+strings.Join(c.values, "|"))
	return fmt.Sprintf("%.5x", h.Sum(nil))
}

func NewInflux(options map[string]string) *Influx {
	optionsFull := map[string]string{
		"url":      "localhost:8086",
		"database": "telegraf",
		"interval": "5",
		"alert":    "",
	}
	for k, v := range options {
		optionsFull[k] = v
	}

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:    optionsFull["url"],
		Timeout: 15 * time.Second,
	})
	if err != nil {
		log.Fatalln("influx: ", err)
	}

	return &Influx{
		client:    c,
		options:   optionsFull,
		templates: make(map[string]*Template),
		checks:    make([]*Check, 0),
	}
}

func (i *Influx) AddTemplate(template *Template) {
	i.templates[template.Name] = template
}

func (i *Influx) AddCheck(check *Check) {
	i.checks = append(i.checks, check)
}

/*
 * Executes query and returns first column as string or float.
 * Returns error if data can't be converted to string or float.
 */
func (i *Influx) querySingleColumn(sql string) (interface{}, error) {
	q := client.Query{
		Command:  sql,
		Database: i.options["database"],
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

/*
 * Executes query and returns first influx result.
 */
func (i *Influx) queryMultipleColumns(sql string) (*client.Result, error) {
	q := client.Query{
		Command:  sql,
		Database: i.options["database"],
	}

	res, err := i.client.Query(q)

	if err != nil {
		return nil, err
	}

	if res.Error() != nil {
		return nil, res.Error()
	}

	//spew.Dump(res.Results)

	return &res.Results[0], nil // return first result
}

/*
 * Monitor interface implementation.
 */
func (i *Influx) GetName() string {
	return "influx"
}

/*
 * Monitor interface implementation.
 * Implements main checking loop for influx.
 */
func (i *Influx) RunWith(notifer *Notifer) {
	interval, err := strconv.Atoi(i.options["interval"])

	if err != nil {
		log.WithFields(log.Fields{"value": i.options["interval"]}).Fatal("influx: wrong 'interval' value")
	}

	i.notifer = notifer
	i.initTriggers(interval)

	for {
		log.Info("influx: check loop...")

		for _, check := range i.checks {

			log.WithFields(log.Fields{"info": check.info}).Debug("influx: next check")

			sql := i.getSqlForCheck(check)
			log.WithFields(log.Fields{"sql": strings.TrimSpace(sql)}).Debug("influx: executing sql")

			value, err := i.querySingleColumn(sql)
			if err != nil {
				log.Error("influx: error during query execution: ", err)
				continue
			}

			log.WithFields(log.Fields{"value": value}).Debug("influx: sending value to trigger")

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

/*
 * Prepare trigger's callback for every check.
 */
func (i *Influx) initTriggers(interval int) {
	channel := i.options["alert"]

	for _, check := range i.checks {
		_check := check // catch var for closure

		// assign new callback-closure
		check.trigger.callback = func(state *State, lastValue interface{}) error {
			details := i.getDetailsForCheck(_check)
			details["value"] = fmt.Sprintf("%v", lastValue)
			details["template"] = _check.template

			sql := i.getSqlForCheck(_check)

			var body string
			switch state.Name {
			case "good":
				body = fmt.Sprintf("Query is good more than %d sec. ```%s```", interval*state.Cycles, sql)
			case "warn":
				body = fmt.Sprintf("*WARN:* query has bad value for more than %d sec. ```%s```", interval*state.Cycles, sql)
			case "crit":
				body = fmt.Sprintf("*CRITICAL:* query has bad value for more than %d sec. ```%s```", interval*state.Cycles, sql)
			}

			msg := Message{
				IconUrl: "https://aperogeek.fr/wp-content/uploads/2017/04/influx_logo.png", // TODO: replace
				From:    "influx",
				Title:   fmt.Sprintf("QUERY: *%s* in %s state", _check.info, strings.ToUpper(state.Name)),
				Body:    body,
				Details: details,
			}

			msg.ParseLevel(state.Name)

			if msg.Level != MSG_LVL_GOOD {
				preview := i.getPreview(&msg, _check)
				if preview == "" {
					msg.Body += "\n`no preview query available`\n"
				} else {
					msg.Body += "\n*preview query:*\n" + preview
				}
			}

			switch msg.Level {
			case MSG_LVL_GOOD:
				i.notifer.Resolve(_check.GetReportId())
			default:
				i.notifer.Report(_check.GetReportId(), msg)
			}

			i.notifer.Notify(state.Name, channel, msg)

			return nil
		}
	}
}

/*
 * Executes and returns formatted output of preview SQL query.
 * Returns empty string preview query was not provided in config file.
 * This method will retry 5 times before fail during influx preview querying.
 */
func (i *Influx) getPreview(msg *Message, check *Check) string {
	log.WithFields(log.Fields{"check": check.info}).Info("influx: executing preview query")

	var preview string
	sql := i.getSqlPreviewForCheck(check)

	if sql == "" {
		return ""
	}

	for try := 0; ; try++ {
		res, err := i.queryMultipleColumns(sql)

		if err == nil {
			if len(res.Series) == 0 {
				return "```<empty dataset>```"
			}

			ser := res.Series[0]

			columns := strings.Join(ser.Columns, ", ")
			table := make([]string, 0)

			for nrow, row := range ser.Values {
				if nrow > 5 {
					table = append(table, "... too many lines in output ...")
					break
				}

				values := make([]string, 0)
				for _, col := range row {
					values = append(values, fmt.Sprintf("%s", col))
				}
				table = append(table, strings.Join(values, ", "))
			}

			preview = fmt.Sprintf("```%s\n%s```", columns, strings.Join(table, "\n"))
			break
		}

		// TODO: config value for number of retries?
		if try >= 5 {
			log.WithFields(log.Fields{"check": check.info}).Error("influx: executing preview query failed after 5 retries")
			preview = fmt.Sprintf("Influx error: %s", err)
			break
		}
	}

	return preview
}

/*
 * Helper for preparing message details field.
 */
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

/*
 * Finds SQL template for check and renders it.
 */
func (i *Influx) getSqlForCheck(check *Check) string {
	tpl, ok := i.templates[check.template]
	if !ok {
		log.WithFields(log.Fields{"module": "influx"}).Fatalf("influx: missing template '%s'\n", check.template)
	}

	return tpl.Format(check.values...)
}

/*
 * Finds SQL preview template for check and renders it.
 */
func (i *Influx) getSqlPreviewForCheck(check *Check) string {
	tpl, ok := i.templates[check.template]
	if !ok {
		log.WithFields(log.Fields{"module": "influx"}).Fatalf("influx: missing template '%s'\n", check.template)
	}

	return tpl.FormatPreview(check.values...)
}

/*
 * Renders SQL template with values
 */
func (t *Template) Format(values ...string) string {
	return t.formatTemplate(t.body, values)
}

/*
 * Renders preview SQL template with values
 */
func (t *Template) FormatPreview(values ...string) string {
	return t.formatTemplate(t.preview, values)
}

/*
 * Renders provided template.
 */
func (t *Template) formatTemplate(tpl string, values []string) string {
	if len(values) != len(t.args) {
		log.WithFields(log.Fields{"values": values, "args": t.args}).Fatalln("influx: wrong call of template", t.Name, " - wrong amount of arguments")
	}

	res := tpl
	for i, arg := range t.args {
		res = strings.Replace(res, "%"+arg, values[i], -1)
	}

	return strings.TrimSpace(res)
}
