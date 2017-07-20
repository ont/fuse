package main

import "os"
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
    Options  map[string]string
    trigger  *Trigger
}

func NewConsul(services []*Service, defaults map[string]string) *Consul {
    defaultsFull := map[string]string{
        "interval" : "5",
        "good"     : "2",
        "warn"     : "3",
        "crit"     : "5",
        "alert"    : "",
    }
    for k,v := range defaults {
        defaultsFull[k] = v
    }

    config := api.DefaultConfig()

    // TODO: move outside this function
    config.Address = os.Getenv("FUSE_CONSUL_ADDRESS")

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

func (c *Consul) StartChecking(notifer *Notifer){
    interval, err := strconv.Atoi(c.defaults["interval"])

    // TODO: better exit
    if err != nil {
        panic(err)
    }

    c.addTriggers(notifer, interval)

    for {
        //fmt.Println("--------- Checking cycle ---------")
        services, _, err := c.catalog.Services(nil)

        // TODO: handle multiple error (write to log/slack)
        if err != nil {
            continue
        }

        c.checkServices(services)

        time.Sleep(time.Second * time.Duration(interval))
    }
}

func (c *Consul) addTriggers(notifer *Notifer, interval int){
    for _, service := range c.Services {

        good := c.ToInt("good", service)
        warn := c.ToInt("warn", service)
        crit := c.ToInt("crit", service)
        channel := c.ToString("alert", service)

        name := service.Name   // without this assigment closures doesn't catches right value for service name
        service.trigger = NewTrigger(
            good, warn, crit,
            func() error {
                return notifer.Good(channel, name, fmt.Sprintf("Service is online more than %d sec.", interval * good))
            },
            func() error {
                return notifer.Warn(channel, name, fmt.Sprintf("Service is offline more than %d sec.", interval * warn))
            },
            func() error {
                return notifer.Crit(channel, name, fmt.Sprintf("Service is offline more than %d sec.", interval * crit))
            },
        )
    }
}

func (c *Consul) ToInt(name string, service *Service) int {
    str := c.getOptionFor(name, service)
    value, err := strconv.Atoi(str)

    // TODO: better exit message (without stacktrace)
    if err != nil {
        panic(err)
    }

    return value
}

func (c *Consul) ToString(name string, service *Service) string {
    return c.getOptionFor(name, service)
}

func (c *Consul) getOptionFor(name string, service *Service) string {
    str, ok := service.Options[name]
    if !ok {
        str = c.defaults[name]
    }
    return str
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
            service.trigger.Good()
        } else {
            service.trigger.Bad()
        }
    }
}
