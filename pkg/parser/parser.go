package parser

import (
	"errors"
	"regexp"
	"strconv"

	"fuse/pkg/consul"
	"fuse/pkg/domain"
	"fuse/pkg/influx"
	"fuse/pkg/monitor"
	"fuse/pkg/slack"
	"fuse/pkg/twilio"

	log "github.com/sirupsen/logrus"
	. "github.com/yhirose/go-peg"
)

type Option struct {
	Key   string
	Value string
}

type ParseResult struct {
	Alerters map[string]domain.Alerter
	Monitors map[string]monitor.Monitor
	Metrics  map[string]domain.Metric
}

// helper class for parsing
type StateValue struct {
	operator string
	value    interface{}
}

// helper class for parsing
type OptionalAlert struct {
	Name string
}

func Parse(text string) (*ParseResult, error) {
	// remove any comments from config
	re := regexp.MustCompile(`(?m)^\s*#.*$`)
	text = re.ReplaceAllString(text, "")

	val, err := getParser().ParseAndGetValue(text, nil)
	if err != nil {
		return nil, err
	}

	if val, ok := val.(*ParseResult); ok {
		logConfig(val)
		return val, nil
	}

	return nil, errors.New("Error during parsing")
}

func logConfig(conf *ParseResult) {
	names := []string{}
	for k, _ := range conf.Alerters {
		names = append(names, k)
	}
	log.WithField("alerters", names).Info("alerters")

	names = []string{}
	for k, _ := range conf.Monitors {
		names = append(names, k)
	}
	log.WithField("monitors", names).Info("monitors")

	for _, monitor := range conf.Monitors {
		monitor.LogInfo()
	}
}

func getParser() *Parser {
	result := &ParseResult{
		make(map[string]domain.Alerter),
		make(map[string]monitor.Monitor),
		make(map[string]domain.Metric),
	}

	parser, _ := NewParser(`
		CONFIG  ← SECTION+
		SECTION ← SLACK / TWILIO / CONSUL / INFLUX

		# Slack
		SLACK   ← 'slack' '{' OPTION+ '}'

		# Twilio
		TWILIO   ← 'twilio' '{' OPTION+ '}'

		# Consul
		CONSUL  ← 'consul' '{' OPTION+ SERVICE+ '}'
		SERVICE ← 'service' STRING ALERT* TRIGGER?
		ALERT   ← 'alert' '(' < STRING > ')'

		# Influx
		INFLUX   ← 'influx' '{' OPTION+ TEMPLATE+ 'checks' '{' CHECK+ '}' '}'
		TEMPLATE ← 'template' FNAME '(' ARGS ')' '{' BODY '}' ('preview' '{' BODY '}')?
		CHECK    ← FNAME '(' (STRING ',')* STRING ')' 'as' STRING TRIGGER

		# Trigger
		TRIGGER     ← STATE+
		STATE       ← FNAME '(' STATE_VALUE ',' INT ('cycles' / 'cycle') (',' ARG)? ')'
		STATE_VALUE ← STRING / (COMPARATOR FLOAT)
		COMPARATOR  ← < '<=' / '>=' / '<' / '>' / '=' >

		# Basic items
		OPTION  ←  KEY '=' STRING
		STRING  ←  '"' < (!'"' .)+ > '"'

		FNAME   ←  < (![ \n(] .)+ >
		ARG     ←  < (![ ,)] .)+ >  # any chars except space, ',' or ')'
		ARGS    ←  (ARG ',')* ARG
		BODY    ←  < (!'}' .)+ >

		KEY     ←  < (![ =] .)+ >
		VALUE   ←  < (![ \n] .)+ >

		INT     ←  < [0-9]+ >
		FLOAT   ←  < ('-' / '+')? INT ('.' INT)? >

		%whitespace  ←  [ \t\n]*
	`)

	g := parser.Grammar

	g["CONFIG"].Action = func(v *Values, d Any) (Any, error) {
		return result, nil
	}

	g["SLACK"].Action = func(v *Values, d Any) (Any, error) {
		options := parseOptions(v)

		//spew.Dump(options)
		channel, ok := options["channel"]
		if !ok {
			log.Fatal("Slack: 'channel' option is required!")
		}

		token, ok := options["token"]
		if !ok {
			log.Fatal("Slack: 'token' option is required!")
		}

		icon := options["icon_url"]

		//spew.Dump("CREATING SLACK:", channel, token, icon)
		result.Alerters["slack"] = slack.NewSlackClient(channel, token, icon)
		return nil, nil
	}

	g["TWILIO"].Action = func(v *Values, d Any) (Any, error) {
		options := parseOptions(v)

		//spew.Dump(options)
		phoneTo, ok := options["phone_to"]
		if !ok {
			log.Fatal("Twilio: 'phone_to' option is required!")
		}

		phoneFrom, ok := options["phone_from"]
		if !ok {
			log.Fatal("Twilio: 'phone_from' option is required!")
		}

		sid, ok := options["sid"]
		if !ok {
			log.Fatal("Twilio: 'sid' option is required!")
		}

		token, ok := options["token"]
		if !ok {
			log.Fatal("Twilio: 'token' option is required!")
		}

		twimlUrl, ok := options["twiml_url"]
		if !ok {
			log.Fatal("Twilio: 'twiml_url' option is required!")
		}

		//spew.Dump("CREATING TWILIO:", phoneTo, phoneFrom, token, sid, twimlUrl)
		result.Alerters["twilio"] = twilio.NewTwilioClient(phoneTo, phoneFrom, token, sid, twimlUrl)

		return nil, nil
	}

	g["CONSUL"].Action = func(v *Values, d Any) (Any, error) {
		options := parseOptions(v)

		services := make([]*consul.Service, 0, v.Len())
		for _, any := range v.Vs {
			if service, ok := any.(*consul.Service); ok {
				services = append(services, service)
			}
		}

		result.Monitors["consul"] = consul.NewConsul(services, options)
		return nil, nil
	}

	g["SERVICE"].Action = func(v *Values, d Any) (Any, error) {
		//spew.Dump("SERVICE", v)
		service := &consul.Service{
			Name:   v.ToStr(0),
			Alerts: make([]string, 0),
		}

		for i := 1; i < v.Len(); i++ {
			if alert, ok := v.Vs[i].(*OptionalAlert); ok {
				service.Alerts = append(service.Alerts, alert.Name)
			}

			if trigger, ok := v.Vs[i].(*domain.Trigger); ok {
				service.Trigger = trigger
			}
		}

		return service, nil
	}

	g["ALERT"].Action = func(v *Values, d Any) (Any, error) {
		return &OptionalAlert{v.ToStr(0)}, nil
	}

	g["TRIGGER"].Action = func(v *Values, d Any) (Any, error) {
		t := domain.NewTrigger(nil)

		for _, any := range v.Vs {
			if state, ok := any.(*domain.State); ok {
				t.AddState(state)
			}
		}

		return t, nil
	}

	g["STATE"].Action = func(v *Values, d Any) (Any, error) {
		state_value, _ := v.Vs[1].(*StateValue)
		state := &domain.State{
			Name:     v.ToStr(0),
			Operator: state_value.operator,
			Value:    state_value.value,
			Cycles:   v.ToInt(2),
		}

		if v.Len() >= 4 && v.ToStr(3) == "allow_nil" {
			state.AllowNil = true
		}

		return state, nil
	}

	g["STATE_VALUE"].Action = func(v *Values, d Any) (Any, error) {
		//spew.Dump("STATE_VALUE", v)
		if v.Len() == 1 {

			//spew.Dump("string path")
			return &StateValue{
				operator: "=",
				value:    v.ToStr(0),
			}, nil

		} else {
			float, _ := v.Vs[1].(float64)

			return &StateValue{
				operator: v.ToStr(0),
				value:    float,
			}, nil
		}
	}

	g["COMPARATOR"].Action = func(v *Values, d Any) (Any, error) {
		//spew.Dump("COMPARATOR", v.Token())
		return v.Token(), nil
	}

	g["INFLUX"].Action = func(v *Values, d Any) (Any, error) {
		options := parseInfluxOptions(v)
		iflux := influx.NewInflux(options)

		for _, value := range v.Vs {
			if template, ok := value.(*influx.Template); ok {
				iflux.AddTemplate(template)
			} else if check, ok := value.(*influx.Check); ok {
				iflux.AddCheck(check)
			}
		}

		result.Monitors["influx"] = iflux
		result.Metrics["influx"] = influx.NewInfluxMetrics(options)

		return nil, nil
	}

	g["TEMPLATE"].Action = func(v *Values, d Any) (Any, error) {
		args, _ := v.Vs[1].([]string)
		body := v.ToStr(2)

		preview := ""
		if v.Len() > 3 {
			preview = v.ToStr(3)
		}

		return &influx.Template{
			Name:    v.ToStr(0),
			Body:    body,
			Preview: preview,
			Args:    args,
		}, nil
	}

	g["CHECK"].Action = func(v *Values, d Any) (Any, error) {
		values := make([]string, 0, v.Len()-2)
		for i := 1; i < v.Len()-2; i++ {
			values = append(values, v.ToStr(i))
		}

		trigger, _ := v.Vs[v.Len()-1].(*domain.Trigger)
		info, _ := v.Vs[v.Len()-2].(string)

		return &influx.Check{
			Template: v.ToStr(0),
			Info:     info,
			Values:   values,
			Trigger:  trigger,
		}, nil
	}

	g["ARG"].Action = func(v *Values, d Any) (Any, error) {
		//spew.Dump("KEY", v.Token())
		return v.Token(), nil
	}

	g["ARGS"].Action = func(v *Values, d Any) (Any, error) {
		values := make([]string, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			values = append(values, v.ToStr(i))
		}
		return values, nil
	}

	g["OPTION"].Action = func(v *Values, d Any) (Any, error) {
		return &Option{
			Key:   v.ToStr(0),
			Value: v.ToStr(1),
		}, nil
	}

	g["KEY"].Action = func(v *Values, d Any) (Any, error) {
		//spew.Dump("KEY", v.Token())
		return v.Token(), nil
	}

	g["VALUE"].Action = func(v *Values, d Any) (Any, error) {
		//spew.Dump("VALUE", v.Token())
		return v.Token(), nil
	}

	g["STRING"].Action = func(v *Values, d Any) (Any, error) {
		return v.Token(), nil
	}

	g["FNAME"].Action = func(v *Values, d Any) (Any, error) {
		//spew.Dump("FNAME", v.Token())
		return v.Token(), nil
	}

	g["BODY"].Action = func(v *Values, d Any) (Any, error) {
		//spew.Dump("BODY", v.Token())
		return v.Token(), nil
	}

	g["FLOAT"].Action = func(v *Values, d Any) (Any, error) {
		return strconv.ParseFloat(v.Token(), 64)
	}

	g["INT"].Action = func(v *Values, d Any) (Any, error) {
		//spew.Dump("INT", v.Token())
		return strconv.Atoi(v.Token())
	}

	return parser
}

func parseOptions(v *Values) map[string]string {
	options := make(map[string]string)
	for _, any := range v.Vs {
		if option, ok := any.(*Option); ok {
			options[option.Key] = option.Value
		}
	}
	return options
}

func parseInfluxOptions(v *Values) influx.InfluxOptions {
	options := influx.DefaultInfluxOptions()
	for _, any := range v.Vs {
		if option, ok := any.(*Option); ok {
			switch option.Key {
			case "url":
				options.Address = option.Value
			case "database":
				options.Database = option.Value
			case "retention_policy":
				options.RetentionPolicy = option.Value
			case "interval":
				interval, err := strconv.Atoi(option.Value)
				if err != nil {
					log.Fatalln("influx: wrong format for interval: ", err)
				}

				options.Interval = interval
			case "alert":
				options.Alert = option.Value
			}
		}
	}
	return options
}
