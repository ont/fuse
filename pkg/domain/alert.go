package domain

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

type Notifer struct {
	Alerters map[string]Alerter
	Metrics  map[string]Metric
}

const (
	MSG_LVL_UNKN = iota
	MSG_LVL_GOOD = iota
	MSG_LVL_WARN = iota
	MSG_LVL_CRIT = iota
)

type Message struct {
	// message warn level - MSG_GOOD || MSG_WARN || MSG_CRIT
	Level int

	// context info (consul/influx/...)
	IconUrl string
	From    string

	// message
	Title string
	Body  string

	// additional info as field-value pairs
	Details map[string]string
	Args    map[string]interface{}
}

type Alerter interface {
	GetName() string

	Good(msg Message) error
	Warn(msg Message) error
	Crit(msg Message) error

	Report(reportId string, msg Message) error
	Resolve(reportId string) error

	ConfigureHTTP()
}

func NewNotifer() *Notifer {
	return &Notifer{
		Alerters: make(map[string]Alerter),
		Metrics:  make(map[string]Metric),
	}
}

func (n *Notifer) AddAlerter(channel string, alerter Alerter) {
	n.Alerters[channel] = alerter
}

func (n *Notifer) AddMetric(name string, metric Metric) {
	n.Metrics[name] = metric
}

func (n *Notifer) AlerterExists(name string) bool {
	_, ok := n.Alerters[name]
	return ok
}

func (n *Notifer) Notify(level string, channels interface{}, msg Message) {
	switch level {
	case "good":
		n.Good(channels, msg)
	case "warn":
		n.Warn(channels, msg)
	case "crit":
		n.Crit(channels, msg)
	default:
		log.WithField("level", level).Warn("alert: unknown level, sending as warn")
		n.Warn(channels, msg)
	}
}

func (n *Notifer) Good(channels interface{}, msg Message) {
	msg.Level = MSG_LVL_GOOD
	n.notifyOneOrMany(channels, func(alerter Alerter) error {
		log.WithField("alerter", alerter.GetName()).Info("alert: send Good message")
		log.WithField("msg", msg).Debug("alert: message")

		err := alerter.Good(msg)
		if err != nil {
			log.WithError(err).Error("alert: error during sending")
		}
		return err
	})

	n.sendMetrics(msg)
}

func (n *Notifer) Warn(channels interface{}, msg Message) {
	msg.Level = MSG_LVL_WARN
	n.notifyOneOrMany(channels, func(alerter Alerter) error {
		log.WithField("channel", alerter.GetName()).Info("alert: send Warn message")
		log.WithField("msg", msg).Debug("alert: message")

		err := alerter.Warn(msg)
		if err != nil {
			log.WithError(err).Error("alert: error during sending")
		}
		return err
	})

	n.sendMetrics(msg)
}

func (n *Notifer) Crit(channels interface{}, msg Message) {
	msg.Level = MSG_LVL_CRIT
	n.notifyOneOrMany(channels, func(alerter Alerter) error {
		log.WithField("channel", alerter.GetName()).Info("alert: send Crit message")
		log.WithField("msg", msg).Debug("alert: message")

		err := alerter.Crit(msg)
		if err != nil {
			log.WithError(err).Error("alert: error during sending")
		}
		return err
	})

	n.sendMetrics(msg)
}

func (n *Notifer) Start() {
	for name, alerter := range n.Alerters {
		name, alerter := name, alerter
		log.WithField("name", name).Info("notifer: configuring alerter")
		alerter.ConfigureHTTP()
	}

	// TODO: configurable port?
	go func() {
		if err := http.ListenAndServe(":7777", nil); err != nil {
			log.WithError(err).Fatal("notifer: can't start http listener")
		}
	}()
}

func (n *Notifer) Report(reportId string, msg Message) {
	for name, alerter := range n.Alerters {
		if err := alerter.Report(reportId, msg); err != nil {
			log.WithField("alerter", name).Error("notifer: error during sending report to alerter: ", err)
		}
	}
}

func (n *Notifer) Resolve(reportId string) {
	for name, alerter := range n.Alerters {
		if err := alerter.Resolve(reportId); err != nil {
			log.WithField("alerter", name).Error("notifer: error during resolve report in alerter: ", err)
		}
	}
}

func (n *Notifer) notifyOneOrMany(channels interface{}, callback func(Alerter) error) {
	if channels, ok := channels.([]string); ok {
		for _, channel := range channels {
			n.notifyChannel(channel, callback)
		}
	}

	if channel, ok := channels.(string); ok {
		n.notifyChannel(channel, callback)
	}
}

func (n *Notifer) notifyChannel(channel string, callback func(Alerter) error) {
	alerter, ok := n.Alerters[channel]
	if !ok {
		log.WithField("channel", channel).Error("channel not found")
		return
	}

	if err := callback(alerter); err != nil {
		log.WithError(err).WithField("channel", channel).Error("error during sending to alerter")
	}
}

func (n *Notifer) sendMetrics(msg Message) {
	for channel, metric := range n.Metrics {
		if err := metric.Save(msg); err != nil {
			log.WithError(err).WithField("channel", channel).Error("error sending metric")
		}
	}
}

func (m *Message) ParseLevel(level string) {
	switch level {
	case "good":
		m.Level = MSG_LVL_GOOD
	case "warn":
		m.Level = MSG_LVL_WARN
	case "crit":
		m.Level = MSG_LVL_CRIT
	default:
		m.Level = MSG_LVL_UNKN
	}
}

func (m *Message) LevelToStr() string {
	switch m.Level {
	case MSG_LVL_CRIT:
		return "crit"
	case MSG_LVL_GOOD:
		return "good"
	case MSG_LVL_WARN:
		return "warn"
	default:
		return "unknown"
	}
}
