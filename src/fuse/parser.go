package main

import (
    "strconv"
    "errors"
    //"github.com/davecgh/go-spew/spew"
    . "github.com/yhirose/go-peg"
)

type Option struct {
    Key    string
    Value  string
}

type ParseResult struct {
    Alerters map[string]Alerter
    Monitors map[string]Monitor
}

// helper class for parsing
type StateValue struct {
    operator string
    value    interface{}
}

func Parse(text string) (*ParseResult, error) {
    val, err := getParser().ParseAndGetValue(text, nil)
    if err != nil {
        return nil, err
    }

    if val, ok := val.(*ParseResult); ok {
        return val, nil
    }

    return nil, errors.New("Error during parsing")
}

func getParser() *Parser {
    result := &ParseResult{
        make(map[string]Alerter),
        make(map[string]Monitor),
    }

    parser, _ := NewParser(`
        CONFIG  ← SECTION+
        SECTION ← CONSUL / SLACK / INFLUX

        SLACK   ← 'slack' '{' OPTION+ '}'

        # Consul
        CONSUL  ← 'consul' '{' OPTION+ SERVICE+ '}'
        SERVICE ← 'service' STRING TRIGGER?

        # Influx
        INFLUX   ← 'influx' '{' OPTION+ TEMPLATE+ 'checks' '{' CHECK+ '}' '}'
        TEMPLATE ← 'template' FNAME '(' ARGS ')' '{' BODY '}' ('preview' '{' BODY '}')?
        CHECK    ← FNAME '(' (STRING ',')* STRING ')' 'as' STRING TRIGGER

        # Trigger
        TRIGGER     ← STATE+
        STATE       ← FNAME '(' STATE_VALUE ',' INT ('cycles' / 'cycle') ')'
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
            panic("Slack: 'channel' option is required!")
        }

        token, ok := options["token"]
        if !ok {
            panic("Slack: 'token' option is required!")
        }

        icon := options["icon_url"]

        //spew.Dump("CREATING SLACK:", channel, token, icon)
        result.Alerters["slack"] = NewSlackClient(channel, token, icon)
        return nil, nil
    }

    g["CONSUL"].Action = func(v *Values, d Any) (Any, error) {
        options := parseOptions(v)

        services := make([]*Service, 0, v.Len())
        for _, any := range v.Vs {
            if service, ok := any.(*Service); ok {
                services = append(services, service)
            }
        }

        result.Monitors["consul"] = NewConsul(services, options)
        return nil, nil
    }

    g["SERVICE"].Action = func(v *Values, d Any) (Any, error) {
        //spew.Dump("SERVICE", v)
        service := &Service{
            Name: v.ToStr(0),
        }
        if v.Len() > 1 {
            service.trigger, _ = v.Vs[1].(*Trigger)
        }
        return service, nil
    }

    g["TRIGGER"].Action = func(v *Values, d Any) (Any, error) {
        t := NewTrigger(nil)

        for _, any := range v.Vs {
            if state, ok := any.(*State); ok {
                t.AddState(state)
            }
        }

        return t, nil
    }

    g["STATE"].Action = func(v *Values, d Any) (Any, error) {
        //spew.Dump("STATE", v)
        state_value, _ := v.Vs[1].(*StateValue)
        return &State{
            Name: v.ToStr(0),
            operator: state_value.operator,
            value: state_value.value,
            Cycles: v.ToInt(2),
        }, nil
    }

    g["STATE_VALUE"].Action = func(v *Values, d Any) (Any, error) {
        //spew.Dump("STATE_VALUE", v)
        if v.Len() == 1 {

            //spew.Dump("string path")
            return &StateValue{
                operator: "=",
                value: v.ToStr(0),
            }, nil

        } else {
            float, _ := v.Vs[1].(float64)

            return &StateValue{
                operator: v.ToStr(0),
                value: float,
            }, nil
        }
    }

    g["COMPARATOR"].Action = func(v *Values, d Any) (Any, error) {
        //spew.Dump("COMPARATOR", v.Token())
        return v.Token(), nil
    }

    g["INFLUX"].Action = func(v *Values, d Any) (Any, error) {
        options := parseOptions(v)
        influx := NewInflux(options)

        for _, value := range v.Vs {
            if template, ok := value.(*Template); ok {
                influx.AddTemplate(template)
            } else if check, ok := value.(*Check); ok {
                influx.AddCheck(check)
            }
        }

        result.Monitors["influx"] = influx
        return nil, nil
    }

    g["TEMPLATE"].Action = func(v *Values, d Any) (Any, error) {
        args, _ := v.Vs[1].([]string)
        body := v.ToStr(2)

        preview := ""
        if v.Len() > 3 {
            preview = v.ToStr(3)
        }

        return &Template{
            Name: v.ToStr(0),
            body: body,
            preview: preview,
            args: args,
        }, nil
    }

    g["CHECK"].Action = func(v *Values, d Any) (Any, error) {
        values := make([]string, 0, v.Len()-2)
        for i := 1; i < v.Len()-2; i++ {
            values = append(values, v.ToStr(i))
        }

        trigger, _ := v.Vs[v.Len()-1].(*Trigger)
        info, _ := v.Vs[v.Len()-2].(string)

        return &Check{
            template: v.ToStr(0),
            info: info,
            values: values,
            trigger: trigger,
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
            Key: v.ToStr(0),
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
