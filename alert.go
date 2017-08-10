package main

import log "github.com/sirupsen/logrus"

type Notifer struct {
    Alerters map[string]Alerter
}

type Alerter interface {
    Good(title string, msg string, details map[string]string) error
    Warn(title string, msg string, details map[string]string) error
    Crit(title string, msg string, details map[string]string) error
}

func NewNotifer() *Notifer {
    return &Notifer{
        Alerters: make(map[string]Alerter),
    }
}

func (n *Notifer) AddAlerter(channel string, alerter Alerter){
    n.Alerters[channel] = alerter
}

func (n *Notifer) Good(channels interface{}, title string, msg string, details map[string]string) error {
    return unpackChannels(channels, func(channel string) error {
        log.WithFields(log.Fields{"channel": channel}).Info("alert: send Good message")
        log.WithFields(log.Fields{"title": title, "msg": msg, "details": details}).Debug("alert: message")
        err := n.Alerters[channel].Good(title, msg, details)
        if err != nil {
            log.Error("alert: error during sending -", err)
        }
        return err
    })
}

func (n *Notifer) Warn(channels interface{}, title string, msg string, details map[string]string) error {
    return unpackChannels(channels, func(channel string) error {
        log.WithFields(log.Fields{"channel": channel}).Info("alert: send Warn message")
        log.WithFields(log.Fields{"title": title, "msg": msg, "details": details}).Debug("alert: message")
        err := n.Alerters[channel].Warn(title, msg, details)
        if err != nil {
            log.Error("alert: error during sending -", err)
        }
        return err
    })
}

func (n *Notifer) Crit(channels interface{}, title string, msg string, details map[string]string) error {
    return unpackChannels(channels, func(channel string) error {
        log.WithFields(log.Fields{"channel": channel}).Info("alert: send Crit message")
        log.WithFields(log.Fields{"title": title, "msg": msg, "details": details}).Debug("alert: message")
        err := n.Alerters[channel].Crit(title, msg, details)
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
