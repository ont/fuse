package main

import (
    "fmt"
    log "github.com/sirupsen/logrus"
)

type Trigger struct {
    state    *State      // current active state
    alerted  bool        // true - if current state was successfully alerted via callback function
    states   []*State    // set of states to check

    callback func(state *State, lastValue interface{}) error  // callback to call after changing the state
}

type State struct {
    Name        string       // name of state
    Cycles      int          // if counter > Cycles then state considered to be active

    counter     int          // count of successfull consecutive Touch'es
    value       interface{}  // value to compare to in Touch
    operator    string       // type of comparision operation (it is always "=" for strings)
    err         bool         // set to true after comparision error in Touch
}

func NewTrigger(callback func(*State, interface{})error) *Trigger {
    return &Trigger{
        state: nil,
        states: make([]*State, 0),
        callback: callback,
    }
}

func (t *Trigger) AddState(state *State) {
    t.states = append(t.states, state)

    // set first state as active
    if len(t.states) == 1 {
        t.state = state
        t.alerted = true
    }
}

/*
 * Compares "value" with each internal state's value.
 * Type of "value" must be string or float64 (it doesn't convert int to float).
 */
func (t *Trigger) Touch(value interface{}) {
    newState := t.state

    log.WithFields(log.Fields{"value" : value}).Debug("trigger: comparing with value")
    for _, state := range t.states {
        canReset := (state != t.state)  // reset only non-active states
        testOk := state.Touch(value, canReset)

        // current test is passed and state is ready to be active
        if testOk && state.IsReady() {
            newState = state
        }
    }

    t.LogStates()

    if t.state != newState {
        t.state = newState
        log.WithFields(log.Fields{"state" : newState.Name}).Debug("trigger: activating new state")
        t.alerted = false

        // reset all states after switching to new active state
        for _, state := range t.states {
            if state != t.state {
                state.Reset()
            }
        }
    }

    // try to resend on every touch if failed
    if !t.alerted {
        err := t.callback(t.state, value)
        if err != nil {
            log.WithFields(log.Fields{"err" : err}).Debug("trigger: error during calling callback")
        } else {
            t.alerted = true
        }
    }
}

func (t *Trigger) LogStates() {
    msg := ""
    for _, state := range t.states {
        msg += fmt.Sprintf("%s:%d(%d) ", state.Name, state.counter, state.Cycles)
    }
    log.WithFields(log.Fields{"states" : msg}).Debug("trigger: states after touching")
}

/*
 * Increments state's counter if internal test returns "true".
 * Reset counter if it is allowed (canReset = true) and internal test returns "false".
 */
func (s *State) Touch(value interface{}, canReset bool) bool {
    if s.test(value) {
        s.counter += 1
        //log.WithFields(log.Fields{"state" : s.Name, "counter" : s.counter}).Debug("trigger: test successfull, counter++")
        return true
    } else if canReset {
        s.counter = 0
        //log.WithFields(log.Fields{"state" : s.Name, "counter" : s.counter}).Debug("trigger: test failed, reset counter")
    }
    return false
}

/*
 * Resets internal state's counter
 */
func (s *State) Reset() {
    s.counter = 0
    //log.WithFields(log.Fields{"state" : s.Name, "counter" : s.counter}).Debug("trigger: resetting counter by request")
}

/*
 * Checks that internal counter is greater then assigned limit
 */
func (s *State) IsReady() bool {
    return s.counter >= s.Cycles
}

func (s *State) test(value interface{}) bool {
    var res, ok1, ok2, ok3 bool

    s.err = false
    if res, ok1 = s.testString(value); ok1 && res { return true }
    if res, ok2 = s.testFloat(value); ok2 && res { return true }
    if res, ok3 = s.testInt(value); ok3 && res { return true }

    if !ok1 && !ok2 && !ok3 {
        s.err = true

        // TODO: doesn't clutter output during tests
        log.WithFields(log.Fields{
            "state" : s.Name,
            "value" : fmt.Sprintf("%s", value),
            "state_value" : fmt.Sprintf("%s", s.value),
        }).Warn("trigger: wrong comparision")
    }

    return false
}

func (s *State) testString(value interface{}) (bool, bool) {
    tmp, ok1 := value.(string)
    svalue, ok2 := s.value.(string)

    if !ok1 || !ok2 {
        return false, false
    }

    //log.WithFields(log.Fields{"state" : s.Name, "value" : value, "state_value" : s.value}).Debug("trigger: comparing as strings")
    return svalue == tmp, true
}


func (s *State) testInt(value interface{}) (bool, bool) {
    tmp, ok := value.(int)

    if !ok {
        return false, false
    }

    //log.WithFields(log.Fields{"state" : s.Name, "value" : value}).Debug("trigger: comparing int as float")
    return s.testFloat(float64(tmp))
}

func (s *State) testFloat(value interface{}) (bool, bool) {
    tmp, ok1 := value.(float64)
    fvalue, ok2 := s.value.(float64)

    if !ok1 || !ok2 {
        return false, false
    }

    //log.WithFields(log.Fields{"state" : s.Name, "value" : value, "state_value" : s.value, "operator": s.operator}).Debug("trigger: comparing as floats")

    switch s.operator {
        case "=":  return tmp == fvalue, true
        case "<":  return tmp <  fvalue, true
        case ">":  return tmp >  fvalue, true
        case "<=": return tmp <= fvalue, true
        case ">=": return tmp >= fvalue, true
        default:   return tmp == fvalue, true
    }
}
