package main

import (
    "fmt"
    "time"
    "strconv"
    "github.com/hashicorp/consul/api"
    log "github.com/sirupsen/logrus"
)
//import "github.com/davecgh/go-spew/spew"

type Consul struct {
    Services  []*Service

    catalog   *api.Catalog       // consul api client
    notifer   *Notifer           // notifer to send alters to

    options   map[string]string  // TODO: replace with explicit declarations (remove parsing actions from Consul)
}

type Service struct {
    Name     string
    trigger  *Trigger
}

func NewConsul(services []*Service, options map[string]string) *Consul {
    optionsFull := map[string]string{
        "url" : "localhost:8500",
        "interval" : "5",
        "alert"    : "",
    }
    for k,v := range options {
        optionsFull[k] = v
    }

    config := api.DefaultConfig()

    config.Address = optionsFull["url"]
    config.WaitTime = 15 * time.Second

    client, err := api.NewClient(config)

    // TODO: better exit message (without stacktrace)
    if err != nil {
        panic(err)
    }

    catalog := client.Catalog()

    return &Consul{
        catalog: catalog,
        Services: services,
        options: optionsFull,
    }
}

func (c *Consul) GetName() string {
    return "consul"
}

func (c *Consul) RunWith(notifer *Notifer){
    interval, err := strconv.Atoi(c.options["interval"])

    if err != nil {
        log.WithFields(log.Fields{"value" : c.options["interval"]}).Fatal("consul: wrong 'interval' value")
    }

    c.notifer = notifer
    c.addTriggers(interval)

    for {
        log.Info("consul: check loop...")
        services, _, err := c.catalog.Services(nil)

        // TODO: handle multiple error (write to log/slack)
        if err != nil {
            log.Error("consul: error during check cycle", err)
        } else {
            c.checkServices(services)
        }

        time.Sleep(time.Duration(interval) * time.Second)
    }
}

func (c *Consul) addTriggers(interval int){
    channel := c.options["alert"]

    for _, service := range c.Services {
        if service.trigger == nil {
            service.trigger = c.defaultTrigger()
        }

        _name := service.Name
        service.trigger.callback = func(state *State, lastValue interface{}) error {
            var err error
            details := map[string]string{
                "value": fmt.Sprintf("%v", lastValue),
            }

            switch state.Name {
                case "good": err = c.notifer.Good(
                    channel,
                    "SERVICE: " + _name,
                    fmt.Sprintf("Service \"%s\" *is online* more than %d sec.", _name, interval * state.Cycles),
                    details,
                )
                case "warn": err = c.notifer.Warn(
                    channel,
                    "SERVICE: " + _name,
                    fmt.Sprintf("*WARN:* service \"%s\" *is offline* more than %d sec.", _name, interval * state.Cycles),
                    details,
                )
                case "crit": err = c.notifer.Crit(
                    channel,
                    "SERVICE: " + _name,
                    fmt.Sprintf("*CRITICAL:* service \"%s\" *is offline* more than %d sec.", _name, interval * state.Cycles),
                    details,
                )
            }
            return err
        }
    }
}

func (c *Consul) defaultTrigger() *Trigger {
    trigger := NewTrigger(nil)

    trigger.AddState(&State{
        Name: "good",
        Cycles: 5,
        operator: "=",
        value: "online",
    })

    trigger.AddState(&State{
        Name: "warn",
        Cycles: 5,
        operator: "=",
        value: "offline",
    })

    trigger.AddState(&State{
        Name: "crit",
        Cycles: 5,
        operator: "=",
        value: "offline",
    })

    return trigger
}

func (c *Consul) checkServices(consulData map[string][]string){
    for _, service := range c.Services {
        found := false
        for name, _ := range consulData {
            if name == service.Name {
                found = true
            }
        }

        if found {
            service.trigger.Touch("online")
        } else {
            service.trigger.Touch("offline")
        }
    }
}
