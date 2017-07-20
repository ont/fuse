package main

import "sync"

var onceNotifer  sync.Once
var notifer      *Notifer

type Notifer struct {
    Alerters map[string]Alerter
}

type Alerter interface {
    Good(name string, msg string) error
    Warn(name string, msg string) error
    Crit(name string, msg string) error
}

func GetNotifer() *Notifer {
    onceNotifer.Do(func(){
        notifer = &Notifer{
            Alerters: make(map[string]Alerter),
        }
    })
    return notifer
}

func (n *Notifer) AddAlerter(channel string, alerter Alerter){
    n.Alerters[channel] = alerter
}

func (n *Notifer) Good(channels interface{}, name string, msg string) error {
    return unpackChannels(channels, func(channel string) error {
        return n.Alerters[channel].Good(name, msg)
    })
}

func (n *Notifer) Warn(channels interface{}, name string, msg string) error {
    return unpackChannels(channels, func(channel string) error {
        return n.Alerters[channel].Warn(name, msg)
    })
}

func (n *Notifer) Crit(channels interface{}, name string, msg string) error {
    return unpackChannels(channels, func(channel string) error {
        return n.Alerters[channel].Crit(name, msg)
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
