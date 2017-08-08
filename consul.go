package main

import "fmt"
import "time"
import "strconv"
import "github.com/hashicorp/consul/api"
//import "github.com/davecgh/go-spew/spew"

type Consul struct {
    Services  []*Service
    defaults  map[string]string
    catalog   *api.Catalog
}

type Service struct {
    Name     string
    trigger  *Trigger
}

func NewConsul(services []*Service, defaults map[string]string) *Consul {
    defaultsFull := map[string]string{
        "url" : "localhost:8500",
        "interval" : "5",
        "alert"    : "",
    }
    for k,v := range defaults {
        defaultsFull[k] = v
    }

    config := api.DefaultConfig()

    config.Address = defaultsFull["url"]

    client, err := api.NewClient(config)

    // TODO: better exit message (without stacktrace)
    if err != nil {
        panic(err)
    }

    catalog := client.Catalog()

    return &Consul{
        catalog: catalog,
        Services: services,
        defaults: defaultsFull,
    }
}

func (c *Consul) RunWith(notifer *Notifer){
    interval, err := strconv.Atoi(c.defaults["interval"])

    // TODO: better exit
    if err != nil {
        panic(err)
    }

    c.addTriggers(notifer, interval)

    fmt.Println("[i] consul: running...")
    for {
        services, _, err := c.catalog.Services(nil)

        // TODO: handle multiple error (write to log/slack)
        if err != nil {
            fmt.Println("[e] consul: error during check cycle")
        } else {
            c.checkServices(services)
        }

        time.Sleep(time.Second * time.Duration(interval))
    }
}

func (c *Consul) addTriggers(notifer *Notifer, interval int){
    channel := c.defaults["alert"]

    for _, service := range c.Services {
        if service.trigger == nil {
            service.trigger = c.defaultTrigger()
        }

        name := service.Name
        service.trigger.callback = func(state *State) error {
            var err error
            switch state.Name {
                case "good": err = notifer.Good(channel, name, fmt.Sprintf("Service is online more than %d sec.", interval * state.Cycles))
                case "warn": err = notifer.Warn(channel, name, fmt.Sprintf("Service is offline more than %d sec.", interval * state.Cycles))
                case "crit": err = notifer.Crit(channel, name, fmt.Sprintf("Service is offline more than %d sec.", interval * state.Cycles))
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
