package main

import "fmt"

type Notifer struct {
    Alerters map[string]Alerter
}

type Alerter interface {
    Good(name string, msg string) error
    Warn(name string, msg string) error
    Crit(name string, msg string) error
}

func NewNotifer() *Notifer {
    return &Notifer{
        Alerters: make(map[string]Alerter),
    }
}

func (n *Notifer) AddAlerter(channel string, alerter Alerter){
    n.Alerters[channel] = alerter
}

func (n *Notifer) Good(channels interface{}, name string, msg string) error {
    return unpackChannels(channels, func(channel string) error {
        fmt.Println("[i] alert: send Good to", channel)
        err := n.Alerters[channel].Good(name, msg)
        if err != nil {
            fmt.Println("[!] alert: error during sending -", err)
        }
        return err
    })
}

func (n *Notifer) Warn(channels interface{}, name string, msg string) error {
    return unpackChannels(channels, func(channel string) error {
        fmt.Println("[i] alert: send Warn to", channel)
        err := n.Alerters[channel].Warn(name, msg)
        if err != nil {
            fmt.Println("[!] alert: error during sending -", err)
        }
        return err
    })
}

func (n *Notifer) Crit(channels interface{}, name string, msg string) error {
    return unpackChannels(channels, func(channel string) error {
        fmt.Println("[i] alert: send Crit to", channel)
        err := n.Alerters[channel].Crit(name, msg)
        if err != nil {
            fmt.Println("[!] alert: error during sending -", err)
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
