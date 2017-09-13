package main

import (
    "io"
    "fmt"
    "time"
    "strconv"
    "crypto/md5"
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


func (s *Service) GetReportId() string {
    h := md5.New()
    io.WriteString(h, s.Name)
    return fmt.Sprintf("%.5x", h.Sum(nil))
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
            log.Error("consul: error during check cycle: ", err)
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

        // lock var for closure function
        _service := service

        service.trigger.callback = func(state *State, lastValue interface{}) error {
            var body string
            var title string
            switch state.Name {
                case "good":
                    title = fmt.Sprintf("SERVICE: *%s* in GOOD state", _service.Name)
                    body = fmt.Sprintf("Service `%s` is online more than %d sec.", _service.Name, interval * state.Cycles)
                case "warn":
                    title = fmt.Sprintf("SERVICE: *%s* in WARN state", _service.Name)
                    body = fmt.Sprintf("Service \"%s\" is offline more than %d sec.", _service.Name, interval * state.Cycles)
                case "crit":
                    title = fmt.Sprintf("SERVICE: *%s* in CRITICAL state", _service.Name)
                    body = fmt.Sprintf("Service \"%s\" is offline more than %d sec.", _service.Name, interval * state.Cycles)
                default:
                    title = fmt.Sprintf("SERVICE: *%s* in WARN state", _service.Name)
                    body = fmt.Sprintf("Service \"%s\" is offline more than %d sec.", _service.Name, interval * state.Cycles)
            }

            switch state.Name {
                case "good":
                    c.notifer.Resolve(_service.GetReportId())
                default:
                    c.notifer.Report(
                        _service.GetReportId(),
                        Message{
                            Title: title,
                            Body: body,
                        },
                    )
            }

            err := c.notifer.Notify(
                state.Name,  // notify level
                channel,
                Message{
                    IconUrl: "https://pbs.twimg.com/media/C5SO5KRVcAA6Ag6.png",  // TODO: replace
                    From: "consul",
                    Title: title,
                    Body: body,
                },
            )
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
        Cycles: 10,
        operator: "=",
        value: "offline",
    })

    return trigger
}

func (c *Consul) checkServices(consulData map[string][]string){
    for _, service := range c.Services {
        log.WithFields(log.Fields{"service" : service.Name}).Debug("consul : checking service")

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
