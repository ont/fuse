package main

type Trigger struct {
    good     int     // minimal cycle count to consider "good" state
    warn     int     // minimal cycle count to consider "warn" state
    crit     int     // minimal cycle count to consider "crit" state
    state    string  // last state ("good", "warn" or "crit"); TODO: enums
    cycles   int     // .. how long trigger in this state (in cycles)
    alerted  bool    // does alert for current state was send successfully?

    // callback to call when state is changed
    goodCallback func() error
    warnCallback func() error
    critCallback func() error
}

func NewTrigger(good int, warn int, crit int,
                goodCallback func() error,
                warnCallback func() error,
                critCallback func() error) *Trigger {

    return &Trigger{
        good: good,
        warn: warn,
        crit: crit,
        goodCallback: goodCallback,
        warnCallback: warnCallback,
        critCallback: critCallback,
    }
}


func (t *Trigger) Good() {
    if t.state == "good" {
        t.cycles += 1

        if t.cycles >= t.good && !t.alerted {
            err := t.goodCallback()
            if err == nil {
                t.alerted = true
            }
        }

    } else {
        t.cycles = 0

        if t.state == "pre-warn" {
            t.alerted = true
        } else {
            t.alerted = false
        }

        t.state = "good"
    }
}

func (t *Trigger) Bad() {
    newState := t.state

    if t.state == "good" {
        t.cycles = 0
    } else {
        t.cycles += 1
    }

    if t.cycles < t.warn {
        newState = "pre-warn"
    } else if t.cycles >= t.warn && t.cycles < t.crit {
        newState = "warn"
    } else if t.cycles >= t.crit {
        newState = "crit"
    }

    if t.state != newState {
        t.alerted = false

        var err error

        switch newState {
        case "pre-warn":
            err = nil
        case "warn":
            err = t.warnCallback()
        case "crit":
            err = t.critCallback()
        }

        if err == nil {
            t.alerted = true
        }

        t.state = newState
    }
}
