package main

import (
	"crypto/md5"
	"fmt"
	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	"io"
	"strconv"
	"strings"
	"time"
)

//import "github.com/davecgh/go-spew/spew"

type Consul struct {
	Services []*Service

	client  *api.Client // consul api client
	notifer *Notifer    // notifer to send alters to

	options map[string]string // TODO: replace with explicit declarations (remove parsing actions from Consul)
}

type Service struct {
	Name    string
	trigger *Trigger
}

func (s *Service) GetReportId() string {
	h := md5.New()
	io.WriteString(h, s.Name)
	return fmt.Sprintf("%.5x", h.Sum(nil))
}

func NewConsul(services []*Service, options map[string]string) *Consul {
	optionsFull := map[string]string{
		"url":      "localhost:8500",
		"interval": "5",
		"alert":    "",
	}
	for k, v := range options {
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

	return &Consul{
		client:   client,
		Services: services,
		options:  optionsFull,
	}
}

func (c *Consul) GetName() string {
	return "consul"
}

func (c *Consul) RunWith(notifer *Notifer) {
	interval, err := strconv.Atoi(c.options["interval"])

	if err != nil {
		log.WithFields(log.Fields{"value": c.options["interval"]}).Fatal("consul: wrong 'interval' value")
	}

	c.notifer = notifer
	c.addTriggers(interval)

	for {
		log.Info("consul: check loop...")
		c.checkServices()
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func (c *Consul) addTriggers(interval int) {
	channel := c.options["alert"]

	for _, service := range c.Services {
		if service.trigger == nil {
			service.trigger = c.defaultTrigger()
		}

		// lock var for closure function
		_service := service

		service.trigger.callback = func(state *State, lastValue interface{}) error {
			var alive string
			if state.Name == "good" {
				alive = "online"
			} else {
				alive = "offline"
			}

			title := fmt.Sprintf("SERVICE: *%s* in %s state", _service.Name, strings.ToUpper(state.Name))
			body := fmt.Sprintf("Service \"%s\" is %s more than %d sec.", _service.Name, alive, interval*state.Cycles)

			msg := Message{
				IconUrl: "https://pbs.twimg.com/media/C5SO5KRVcAA6Ag6.png", // TODO: replace
				From:    "consul",
				Title:   title,
				Body:    body,
			}

			msg.ParseLevel(state.Name)

			switch state.Name {
			case "good":
				c.notifer.Resolve(_service.GetReportId())
			default:
				c.notifer.Report(_service.GetReportId(), msg)
			}

			return c.notifer.Notify(state.Name, channel, msg)

		}
	}
}

func (c *Consul) defaultTrigger() *Trigger {
	trigger := NewTrigger(nil)

	trigger.AddState(&State{
		Name:     "good",
		Cycles:   5,
		operator: "=",
		value:    "online",
	})

	trigger.AddState(&State{
		Name:     "warn",
		Cycles:   5,
		operator: "=",
		value:    "offline",
	})

	trigger.AddState(&State{
		Name:     "crit",
		Cycles:   10,
		operator: "=",
		value:    "offline",
	})

	return trigger
}

func (c *Consul) checkServices() {
	for _, service := range c.Services {
		c.checkService(service)
	}
}

func (c *Consul) checkService(service *Service) {
	log.WithFields(log.Fields{"service": service.Name}).Debug("consul : checking service")

	sinfos, _, err := c.client.Health().Service(
		service.Name, // name of service
		"",           // optinal tag for filtering
		false,        // passingOnly - passing all health checks
		nil,          // QueryOptions
	)

	if err != nil {
		log.WithError(err).WithField("service", service.Name).Error("error during api call to consul for service")
		return
	}

	passing := true

	if len(sinfos) == 0 {
		passing = false // too bad, no services with this name are registered in consul
	}

Loop:
	for _, sinfo := range sinfos {
		for _, check := range sinfo.Checks {
			if check.Status != "passing" {
				passing = false
				break Loop // one of found services doesn't pass health check
			}
		}
	}

	if passing {
		service.trigger.Touch("online")
	} else {
		service.trigger.Touch("offline")
	}
}
