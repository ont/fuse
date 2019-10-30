package influx

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/influxdata/influxdb/client/v2"
	log "github.com/sirupsen/logrus"

	"fuse/pkg/domain"
)

type Influx struct {
	client  client.Client   // influx api client
	notifer *domain.Notifer // notifer to send alters to

	options   InfluxOptions
	templates map[string]*Template
	checks    []*Check
}

type Template struct {
	Name    string
	Body    string
	Preview string
	Args    []string
}

type Check struct {
	Template string // name of template
	Info     string // info string for alert message
	Values   []string
	Trigger  *domain.Trigger
}

// TODO: add constructor and save calculation into cache-field
func (c *Check) GetReportId() string {
	h := md5.New()
	io.WriteString(h, c.Template+"|"+c.Info+"|"+strings.Join(c.Values, "|"))
	return fmt.Sprintf("%.5x", h.Sum(nil))
}

func NewInflux(options InfluxOptions) *Influx {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:    options.Address,
		Timeout: 15 * time.Second,
	})
	if err != nil {
		log.Fatalln("influx: ", err)
	}

	return &Influx{
		client:    c,
		options:   options,
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
		Database: i.options.Database,
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
		Database: i.options.Database,
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
func (i *Influx) RunWith(notifer *domain.Notifer) {
	i.notifer = notifer
	i.setupTriggers(i.options.Interval)

	for {
		log.Info("influx: check loop...")

		for _, check := range i.checks {

			log.WithFields(log.Fields{"info": check.Info}).Debug("influx: next check")

			sql := i.getSqlForCheck(check)
			log.WithFields(log.Fields{"sql": strings.TrimSpace(sql)}).Debug("influx: executing sql")

			value, err := i.querySingleColumn(sql)
			if err != nil {
				log.Error("influx: error during query execution: ", err)
				continue
			}

			log.WithFields(log.Fields{"value": value}).Debug("influx: sending value to trigger")

			//if value == nil {
			//	check.Trigger.Fail("<nil value>")
			//	log.Debug("influx: failing trigger due to 'nil' value")
			//	continue
			//}

			check.Trigger.Touch(value)
		}

		time.Sleep(time.Duration(i.options.Interval) * time.Second)
	}
}

/*
 * Prepare trigger's callback for every check.
 */
func (i *Influx) setupTriggers(interval int) {
	channel := i.options.Alert

	for _, check := range i.checks {
		check.Trigger.SetupNilStates()

		_check := check // catch var for closure

		// assign new callback-closure
		check.Trigger.Callback = func(state *domain.State, lastValue interface{}) error {
			args := i.getArgsForCheck(_check)

			details := map[string]string{
				"value":    fmt.Sprintf("%v", lastValue),
				"template": _check.Template,
			}

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

			msg := domain.Message{
				IconUrl: "https://aperogeek.fr/wp-content/uploads/2017/04/influx_logo.png", // TODO: replace
				From:    "influx",
				Title:   fmt.Sprintf("QUERY: *%s* in %s state", _check.Info, strings.ToUpper(state.Name)),
				Body:    body,
				Details: details,
				Args:    args,
			}

			msg.ParseLevel(state.Name)

			if msg.Level != domain.MSG_LVL_GOOD {
				preview := i.getPreview(&msg, _check)
				if preview == "" {
					msg.Body += "\n`no preview query available`\n"
				} else {
					msg.Body += "\n*preview query:*\n" + preview
				}
			}

			switch msg.Level {
			case domain.MSG_LVL_GOOD:
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
func (i *Influx) getPreview(msg *domain.Message, check *Check) string {
	log.WithFields(log.Fields{"check": check.Info}).Info("influx: executing preview query")

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
			log.WithFields(log.Fields{"check": check.Info}).Error("influx: executing preview query failed after 5 retries")
			preview = fmt.Sprintf("Influx error: %s", err)
			break
		}
	}

	return preview
}

/*
 * Helper for preparing message details field.
 */
func (i *Influx) getArgsForCheck(check *Check) map[string]interface{} {
	tpl, _ := i.templates[check.Template]

	args := make(map[string]interface{})
	for i, arg := range tpl.Args {
		args[arg] = check.Values[i]
	}

	return args
}

/*
 * Finds SQL template for check and renders it.
 */
func (i *Influx) getSqlForCheck(check *Check) string {
	tpl, ok := i.templates[check.Template]
	if !ok {
		log.WithFields(log.Fields{"module": "influx"}).Fatalf("influx: missing template '%s'\n", check.Template)
	}

	return tpl.Format(check.Values...)
}

/*
 * Finds SQL preview template for check and renders it.
 */
func (i *Influx) getSqlPreviewForCheck(check *Check) string {
	tpl, ok := i.templates[check.Template]
	if !ok {
		log.WithFields(log.Fields{"module": "influx"}).Fatalf("influx: missing template '%s'\n", check.Template)
	}

	return tpl.FormatPreview(check.Values...)
}

/*
 * Renders SQL template with values
 */
func (t *Template) Format(values ...string) string {
	return t.formatTemplate(t.Body, values)
}

/*
 * Renders preview SQL template with values
 */
func (t *Template) FormatPreview(values ...string) string {
	return t.formatTemplate(t.Preview, values)
}

/*
 * Renders provided template.
 */
func (t *Template) formatTemplate(tpl string, values []string) string {
	if len(values) != len(t.Args) {
		log.WithFields(log.Fields{"values": values, "args": t.Args}).Fatalln("influx: wrong call of template", t.Name, " - wrong amount of arguments")
	}

	res := tpl
	for i, arg := range t.Args {
		res = strings.Replace(res, "%"+arg, values[i], -1)
	}

	return strings.TrimSpace(res)
}

func (i *Influx) LogInfo() {
	log.WithField("monitor", i.GetName()).WithField("amount", len(i.templates)).Info("amount of templates")
	log.WithField("monitor", i.GetName()).WithField("amount", len(i.checks)).Info("amount of checks")

	for _, tpl := range i.templates {
		log.WithField("monitor", i.GetName()).WithField("template", tpl.Name).Info("template")
	}

	for _, chk := range i.checks {
		log.WithField("monitor", i.GetName()).WithField("template", chk.Template).WithField("info", chk.Info).Info("check")
	}
}
