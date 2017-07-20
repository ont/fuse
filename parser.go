package main

import (
    "errors"
    "github.com/davecgh/go-spew/spew"
    . "github.com/yhirose/go-peg"
)

type Option struct {
    Key    string
    Value  string
}

func Parse(text string) (*Consul, error) {
    val, err := getParser().ParseAndGetValue(text, nil)
    if err != nil {
        return nil, err
    }

    if val, ok := val.(*Consul); ok {
        return val, nil
    }

    return nil, errors.New("Result is not *Consul")
}

func getParser() *Parser {
    parser, _ := NewParser(`
        CONFIG  ← SECTION+
        SECTION ← CONSUL / SLACK
        SLACK   ← 'slack' '{' OPTION+ '}'
        CONSUL  ← 'consul' '{' OPTION+ SERVICE+ '}'
        SERVICE ← 'service' '"' STRING '"' ('{' OPTION+ '}')?
        OPTION  ← KEY '=' VALUE
        
        # Basic items
        STRING  ←  < (!'"' .)+ >
        KEY     ←  < (![ =] .)+ >
        VALUE   ←  < (![ \n] .)+ >

        %whitespace  ←  [ \t\n]*
    `)

    g := parser.Grammar

    g["SLACK"].Action = func(v *Values, d Any) (Any, error) {
        options := parseOptions(v)

        spew.Dump(options)
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
        GetNotifer().AddAlerter("slack",
            NewSlackClient(channel, token, icon),
        )

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

        GetFuse().AddMonitor(
            NewConsul(services, options),
        )
        return nil, nil
    }

    g["SERVICE"].Action = func(v *Values, d Any) (Any, error) {
        //spew.Dump("SERVICE", v)
        name := v.ToStr(0)
        options := parseOptions(v)

        return &Service{
            Name: name,
            Options: options,
        }, nil
    }

    g["OPTION"].Action = func(v *Values, d Any) (Any, error) {
        return &Option{
            Key: v.ToStr(0),
            Value: v.ToStr(1),
        }, nil
    }

    g["KEY"].Action = func(v *Values, d Any) (Any, error) {
        spew.Dump("KEY", v.Token())
        return v.Token(), nil
    }

    g["VALUE"].Action = func(v *Values, d Any) (Any, error) {
        spew.Dump("VALUE", v.Token())
        return v.Token(), nil
    }

    g["STRING"].Action = func(v *Values, d Any) (Any, error) {
        return v.Token(), nil
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
