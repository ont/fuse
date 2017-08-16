package main

import log "github.com/sirupsen/logrus"

type Notifer struct {
    Alerters map[string]Alerter
}

type Message struct {
    // context info (consul/influx/...)
    IconUrl  string
    From     string

    // message
    Title    string
    Body     string

    // additional info as field-value pairs
    Details  map[string]string
}

type Alerter interface {
    Good(msg Message) error
    Warn(msg Message) error
    Crit(msg Message) error
}

func NewNotifer() *Notifer {
    return &Notifer{
        Alerters: make(map[string]Alerter),
    }
}

func (n *Notifer) AddAlerter(channel string, alerter Alerter){
    n.Alerters[channel] = alerter
}

func (n *Notifer) AlerterExists(name string) bool {
    _, ok := n.Alerters[name]
    return ok
}

func (n *Notifer) Notify(level string, channels interface{}, msg Message) error {
    switch level {
    case "good":
        return n.Good(channels, msg)
    case "warn":
        return n.Warn(channels, msg)
    case "crit":
        return n.Crit(channels, msg)
    default:
        log.WithFields(log.Fields{"level": level}).Warn("alert: unknown level, sending as warn")
        return n.Warn(channels, msg)
    }
}

func (n *Notifer) Good(channels interface{}, msg Message) error {
    return unpackChannels(channels, func(channel string) error {
        log.WithFields(log.Fields{"channel": channel}).Info("alert: send Good message")
        log.WithFields(log.Fields{"msg": msg}).Debug("alert: message")
        err := n.Alerters[channel].Good(msg)
        if err != nil {
            log.Error("alert: error during sending -", err)
        }
        return err
    })
}

func (n *Notifer) Warn(channels interface{}, msg Message) error {
    return unpackChannels(channels, func(channel string) error {
        log.WithFields(log.Fields{"channel": channel}).Info("alert: send Warn message")
        log.WithFields(log.Fields{"msg": msg}).Debug("alert: message")
        err := n.Alerters[channel].Warn(msg)
        if err != nil {
            log.Error("alert: error during sending -", err)
        }
        return err
    })
}

func (n *Notifer) Crit(channels interface{}, msg Message) error {
    return unpackChannels(channels, func(channel string) error {
        log.WithFields(log.Fields{"channel": channel}).Info("alert: send Crit message")
        log.WithFields(log.Fields{"msg": msg}).Debug("alert: message")
        err := n.Alerters[channel].Crit(msg)
        if err != nil {
            log.Error("alert: error during sending -", err)
        }
        return err
    })
}

func unpackChannels(channels interface{}, callback func(string)error) error {
    var resErr error

    if channels, ok := channels.([]string); ok {
        for _, channel := range channels {
            err := callback(channel)
            if err != nil {
                resErr = err
            }
        }
    }

    if channel, ok := channels.(string); ok {
        err := callback(channel)
        if err != nil {
            resErr = err
        }
    }

    return resErr
}
