package domain

import (
	"fmt"
	log "github.com/sirupsen/logrus"
)

const STATE_CRIT = "crit"

type Trigger struct {
	state  *State   // current active state
	states []*State // set of states to check

	Callback func(state *State, lastValue interface{}) error // callback to call after changing the state
}

type State struct {
	Name   string // name of state
	Cycles int    // if counter > Cycles then state considered to be active

	counter  int         // count of successfull consecutive Touch'es
	Value    interface{} // value to compare to in Touch
	Operator string      // type of comparision operation (it is always "=" for strings)
	err      bool        // set to true after comparision error in Touch

	AllowNil bool // all <nil> values will trigger this state
}

func NewTrigger(callback func(*State, interface{}) error) *Trigger {
	return &Trigger{
		state:    nil,
		states:   make([]*State, 0),
		Callback: callback,
	}
}

func (t *Trigger) AddState(state *State) {
	t.states = append(t.states, state)

	// set first state as active
	if len(t.states) == 1 {
		t.state = state
	}
}

/*
 * Checks that only one state of trigger has AllowNil == true.
 * If no states has AllowNil flag set then if try to set it up for any "crit" state.
 */
func (t *Trigger) SetupNilStates() {
	var critState *State
	cnt := 0

	for _, state := range t.states {
		// TODO: change t.states to map[string]*State with check order in separate field
		if state.Name == STATE_CRIT {
			critState = state
		}

		if state.AllowNil {
			cnt += 1
		}
	}

	switch {
	case cnt > 1:
		log.WithField("cnt", cnt).Fatalln("trigger: only one state of trigger can be with 'allow_nil' option")
	case cnt == 0:
		if critState != nil {
			critState.AllowNil = true
		}
	}
}

/*
 * Compares "value" with each internal state's value.
 * Type of "value" must be string or float64 (it doesn't convert int to float).
 */
func (t *Trigger) Touch(value interface{}) {
	newState := t.state

	log.WithFields(log.Fields{"value": value}).Debug("trigger: comparing with value")
	for _, state := range t.states {
		doReset := (state != t.state) // reset only non-active states
		testOk := state.Touch(value, doReset)

		// current test is passed and state is ready to be active
		if testOk && state.IsReady() {
			newState = state
		}
	}

	t.LogStates()

	if t.state != newState {
		t.activateState(newState, value)
	}

}

/*
 * Fail function immediately switch trigger to STATE_CRIT state
 */
func (t *Trigger) Fail(value interface{}) {
	log.WithField("value", value).Debug("trigger: failing trigger with value")

	var critState *State
	for _, state := range t.states {
		if state.Name == STATE_CRIT {
			critState = state
		}
	}

	if critState == nil {
		log.Error("trigger: can't find STATE_CRIT state inside trigger")
		return
	}

	for _, state := range t.states {
		if state == critState {
			t.activateState(state, value) // switch trigger to crit state
		} else {
			state.Reset() // reset all counters on non-crit states
		}
	}

	t.LogStates()
}

func (t *Trigger) activateState(state *State, value interface{}) {
	if t.state == state {
		log.WithFields(log.Fields{"state": state.Name}).Debug("trigger: state already activated")
		return // nothing to trigger
	}

	t.state = state
	log.WithFields(log.Fields{"state": state.Name}).Debug("trigger: activating new state")

	// reset all states after switching to new active state
	for _, state := range t.states {
		if state != t.state {
			state.Reset()
		}
	}

	// inform via callback function
	err := t.Callback(t.state, value)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Debug("trigger: error during calling callback")
	}
}

func (t *Trigger) LogStates() {
	msg := ""
	for _, state := range t.states {
		msg += fmt.Sprintf("%s:%d(%d) ", state.Name, state.counter, state.Cycles)
	}
	log.WithFields(log.Fields{"states": msg}).Debug("trigger: current trigger states")
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
	if res, ok1 = s.testString(value); ok1 && res {
		return true
	}
	if res, ok2 = s.testFloat(value); ok2 && res {
		return true
	}
	if res, ok3 = s.testInt(value); ok3 && res {
		return true
	}
	if s.testNil(value) {
		return true
	}

	if value != nil && (!ok1 && !ok2 && !ok3) {
		s.err = true

		// TODO: doesn't clutter output during tests
		log.WithFields(log.Fields{
			"state":       s.Name,
			"value":       fmt.Sprintf("%s", value),
			"state_value": fmt.Sprintf("%s", s.Value),
		}).Warn("trigger: wrong comparision")
	}

	return false
}

func (s *State) testString(value interface{}) (bool, bool) {
	tmp, ok1 := value.(string)
	svalue, ok2 := s.Value.(string)

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
	fvalue, ok2 := s.Value.(float64)

	if !ok1 || !ok2 {
		return false, false
	}

	//log.WithFields(log.Fields{"state" : s.Name, "value" : value, "state_value" : s.value, "operator": s.operator}).Debug("trigger: comparing as floats")

	switch s.Operator {
	case "=":
		return tmp == fvalue, true
	case "<":
		return tmp < fvalue, true
	case ">":
		return tmp > fvalue, true
	case "<=":
		return tmp <= fvalue, true
	case ">=":
		return tmp >= fvalue, true
	default:
		return tmp == fvalue, true
	}
}

func (s *State) testNil(value interface{}) bool {
	return s.AllowNil && value == nil
}
