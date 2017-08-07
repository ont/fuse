package main

import (
    "os"
    "fmt"
    "github.com/davecgh/go-spew/spew"
)

type Trigger struct {
    state  *State          // current active state
    states []*State        // set of states to check

    callback func(state *State) error  // callback to call after changing the state
}

type State struct {
    Name        string       // name of state

    counter     int          // count of successfull consecutive Touch'es
    counterMax  int          // if counter > counterMax then state considered to be active
    value       interface{}  // value to compare to in Touch
    operator    string       // type of comparision operation (it is always "=" for strings)
    err         bool         // set to true after comparision error in Touch
}

func NewTrigger(callback func(*State)error) *Trigger {
    return &Trigger{
        state: nil,
        states: make([]*State, 0),
        callback: callback,
    }
}

func (t *Trigger) AddState(state *State) {
    t.states = append(t.states, state)
}

func (t *Trigger) Touch(value interface{}) {
    ni := int(0)
    newState := t.state

    for i, state := range t.states {
        state.Touch(value)

        if state.IsActive() {
            newState = state
            ni = i
        }
    }

    // reset all previous states, only newState must be active
    for i, state := range t.states {
        if i < ni {
            state.Reset()
        }
    }

    if t.state != newState {
        t.state = newState
        t.callback(newState)
    }
}

func (s *State) Touch(value interface{}) {
    if s.test(value) {
        s.counter += 1
    } else {
        s.counter = 0
    }
}

func (s *State) Reset() {
    s.counter = 0
}

func (s *State) IsActive() bool {
    return s.counter >= s.counterMax
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
        fmt.Fprintln(os.Stderr, "[W] wrong comparision:")
        spew.Fdump(os.Stderr, value, s)
    }

    return false
}

func (s *State) testString(value interface{}) (bool, bool) {
    tmp, ok1 := value.(string)
    svalue, ok2 := s.value.(string)

    if !ok1 || !ok2 {
        return false, false
    }

    return svalue == tmp, true
}


func (s *State) testInt(value interface{}) (bool, bool) {
    tmp, ok := value.(int)

    if !ok {
        return false, false
    }

    return s.testFloat(float32(tmp))
}

func (s *State) testFloat(value interface{}) (bool, bool) {
    tmp, ok1 := value.(float32)
    fvalue, ok2 := s.value.(float32)

    if !ok1 || !ok2 {
        return false, false
    }

    switch s.operator {
        case "=":  return tmp == fvalue, true
        case "<":  return tmp <  fvalue, true
        case ">":  return tmp >  fvalue, true
        case "<=": return tmp <= fvalue, true
        case ">=": return tmp >= fvalue, true
        default:   return tmp == fvalue, true
    }
}
