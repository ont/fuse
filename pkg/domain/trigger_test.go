package domain

import "testing"
import "github.com/stretchr/testify/assert"
import log "github.com/sirupsen/logrus"

func TestStateAllowNilValue(t *testing.T) {
	matrix := map[interface{}]map[interface{}]bool{
		"123":        {"123": true, "12": false, 123: false, nil: true},
		float64(123): {"123": false, "12": false, 123: true, float64(123.0): true, nil: true},
	}
	for svalue, tests := range matrix {
		for tvalue, result := range tests {
			s := State{
				Cycles:   1,
				Value:    svalue,
				AllowNil: true,
			}
			s.Touch(tvalue, true)
			assert.Equalf(t, result, s.IsReady(), "Wrong state %#v for comparision %s == %s", s.IsReady(), svalue, tvalue)
		}
	}
}

func TestStateDisallowNilValue(t *testing.T) {
	matrix := map[interface{}]map[interface{}]bool{
		"123":        {"123": true, "12": false, 123: false, nil: false},
		float64(123): {"123": false, "12": false, 123: true, float64(123.0): true, nil: false},
	}
	for svalue, tests := range matrix {
		for tvalue, result := range tests {
			s := State{
				Cycles:   1,
				Value:    svalue,
				AllowNil: false,
			}
			s.Touch(tvalue, true)
			assert.Equalf(t, result, s.IsReady(), "Wrong state %#v for comparision %s == %s", s.IsReady(), svalue, tvalue)
		}
	}
}

func TestAllowNilForCritIsSet(t *testing.T) {
	trig := NewTrigger(nil)

	goodState := &State{
		Name:     "good",
		Cycles:   1,
		Value:    0,
		AllowNil: false,
	}

	warnState := &State{
		Name:     "warn",
		Cycles:   1,
		Value:    5,
		AllowNil: false,
	}

	critState := &State{
		Name:     "crit",
		Cycles:   1,
		Value:    10,
		AllowNil: false,
	}

	trig.AddState(goodState)
	trig.AddState(warnState)
	trig.AddState(critState)

	trig.SetupNilStates()

	assert.Equalf(t, true, critState.AllowNil, "crit state must have AllowNil == true after SetupNilStates()")
	assert.Equalf(t, false, warnState.AllowNil, "warn state must have AllowNil == false after SetupNilStates()")
	assert.Equalf(t, false, goodState.AllowNil, "good state must have AllowNil == false after SetupNilStates()")
}

func TestStateMatrixEq(t *testing.T) {
	matrix := map[interface{}]map[interface{}]bool{
		"123":        {"123": true, "12": false, 123: false},
		float64(123): {"123": false, "12": false, 123: true, float64(123.0): true},
	}
	for svalue, tests := range matrix {
		for tvalue, result := range tests {
			s := State{
				Cycles: 1,
				Value:  svalue,
			}
			s.Touch(tvalue, true)
			assert.Equalf(t, result, s.IsReady(), "Wrong state %#v for comparision %s == %s", s.IsReady(), svalue, tvalue)
		}
	}
}

func TestStateMatrixLt(t *testing.T) {
	matrix := map[interface{}]map[interface{}]bool{
		float64(123): {"123": false, 122: true, 124: false, float64(122.0): true, float64(124.0): false},
		float64(0):   {"123": false, 122: false, 124: false, float64(122.0): false, float64(124.0): false},
	}
	for svalue, tests := range matrix {
		for tvalue, result := range tests {
			s := State{
				Cycles:   1,
				Value:    svalue,
				Operator: "<",
			}
			s.Touch(tvalue, true)
			assert.Equalf(t, result, s.IsReady(), "Wrong state %#v for comparision %s < %s", s.IsReady(), svalue, tvalue)
		}
	}
}

func TestStateMatrixGt(t *testing.T) {
	matrix := map[interface{}]map[interface{}]bool{
		float64(123): {"123": false, 122: false, 124: true, float64(122.0): false, float64(124.0): true},
		float64(0):   {"123": false, 122: true, 124: true, float64(122.0): true, float64(124.0): true},
	}
	for svalue, tests := range matrix {
		for tvalue, result := range tests {
			s := State{
				Cycles:   1,
				Value:    svalue,
				Operator: ">",
			}
			s.Touch(tvalue, true)
			assert.Equalf(t, result, s.IsReady(), "Wrong state %#v for comparision %s > %s", s.IsReady(), svalue, tvalue)
		}
	}
}

func TestStateMatrixLte(t *testing.T) {
	matrix := map[interface{}]map[interface{}]bool{
		float64(123): {"123": false, 122: true, 124: false, float64(122.0): true, float64(124.0): false, 123: true, float64(123): true},
		float64(0):   {"123": false, 122: false, 124: false, float64(122.0): false, float64(124.0): false, 0: true, float64(0): true},
	}
	for svalue, tests := range matrix {
		for tvalue, result := range tests {
			s := State{
				Cycles:   1,
				Value:    svalue,
				Operator: "<=",
			}
			s.Touch(tvalue, true)
			assert.Equalf(t, result, s.IsReady(), "Wrong state %#v for comparision %s <= %s", s.IsReady(), svalue, tvalue)
		}
	}
}

func TestStateMatrixGte(t *testing.T) {
	matrix := map[interface{}]map[interface{}]bool{
		float64(123): {"123": false, 122: false, 124: true, float64(122.0): false, float64(124.0): true, 123: true, float64(123): true},
		float64(0):   {"123": false, 122: true, 124: true, float64(122.0): true, float64(124.0): true, 0: true, float64(0): true},
	}
	for svalue, tests := range matrix {
		for tvalue, result := range tests {
			s := State{
				Cycles:   1,
				Value:    svalue,
				Operator: ">=",
			}
			s.Touch(tvalue, true)
			assert.Equalf(t, result, s.IsReady(), "Wrong state %#v for comparision %s >= %s", s.IsReady(), svalue, tvalue)
		}
	}
}

func TestStateCounter(t *testing.T) {
	s := State{
		Cycles:   3,
		Value:    float64(123),
		Operator: "<",
	}
	s.Touch(0, true)
	assert.False(t, s.IsReady(), "Must be inactive")

	s.Touch(123, true)
	assert.False(t, s.IsReady(), "Must be inactive")

	s.Touch(0, true)
	assert.False(t, s.IsReady(), "Must be inactive")

	s.Touch(0, true)
	assert.False(t, s.IsReady(), "Must be inactive")

	s.Touch(0, true)
	assert.True(t, s.IsReady(), "Must be active")

	s.Touch(123, true)
	assert.False(t, s.IsReady(), "Must be inactive")

	s.Touch(0, true)
	assert.False(t, s.IsReady(), "Must be inactive")
}

func TestTrigger(t *testing.T) {
	trigger := NewTrigger(func(state *State, value interface{}) error {
		return nil
	})

	trigger.AddState(&State{
		Name:     "good",
		Cycles:   2,
		Operator: "=",
		Value:    "online",
	})

	trigger.AddState(&State{
		Name:     "warn",
		Cycles:   2,
		Operator: "=",
		Value:    "offline",
	})

	trigger.AddState(&State{
		Name:     "crit",
		Cycles:   5,
		Operator: "=",
		Value:    "offline",
	})

	log.Info("-----------------------")
	log.SetLevel(log.DebugLevel)
	assert.Equal(t, "good", trigger.state.Name, "Initial state of trigger must be 'good'")

	trigger.Touch("offline")
	assert.Equal(t, "good", trigger.state.Name, "First 'offline' must not change state of trigger")

	trigger.Touch("offline")
	assert.Equal(t, "warn", trigger.state.Name, "Second 'offline' must change state to 'warn")

	trigger.Touch("offline")
	assert.Equal(t, "warn", trigger.state.Name, "We are start counting from the beginning")

	trigger.Touch("offline")
	assert.Equal(t, "warn", trigger.state.Name, "Second 'offline' - state is 'warn'")

	trigger.Touch("offline")
	assert.Equal(t, "warn", trigger.state.Name, "Third 'offline' - state is 'warn'")

	trigger.Touch("offline")
	trigger.Touch("offline")
	assert.Equal(t, "crit", trigger.state.Name, "Now it must be at 'crit' state")

	trigger.Touch("online")
	assert.Equal(t, "crit", trigger.state.Name, "'online' state doesn't immediatelly change trigger state")

	trigger.Touch("offline")
	trigger.Touch("offline")
	assert.Equal(t, "crit", trigger.state.Name, "Trigger's state is still 'crit' and doesn't 'warn'")

	trigger.Touch("offline")
	assert.Equal(t, "crit", trigger.state.Name, "Trigger's state is still 'crit' and doesn't 'warn'")

	trigger.Touch("online")
	trigger.Touch("online")
	assert.Equal(t, "good", trigger.state.Name, "Trigger's state must be 'good' now")

	trigger.Touch("offline")
	assert.Equal(t, "good", trigger.state.Name, "'good' state is stable and doesn't immediatelly switched to 'warn'")

	trigger.Touch("offline")
	assert.Equal(t, "warn", trigger.state.Name, "Second 'offline' must switch state to 'warn'")

	log.SetLevel(log.InfoLevel)
}

func TestTriggerFail(t *testing.T) {
	var callCnt int

	log.SetLevel(log.DebugLevel)

	trigger := NewTrigger(func(state *State, value interface{}) error {
		callCnt++
		return nil
	})

	trigger.Fail("<test value>")

	assert.Equal(t, 0, callCnt, "callback must not be called (no STATE_CRIT state)")

	trigger.AddState(&State{
		Name:     "good",
		Cycles:   2,
		Operator: "=",
		Value:    "online",
	})

	trigger.AddState(&State{
		Name:     "crit",
		Cycles:   5,
		Operator: "=",
		Value:    "offline",
	})

	trigger.Fail("<test value>")
	trigger.Fail("<test value>")

	assert.Equal(t, 1, callCnt, "callback must be triggered only once")

	trigger.Touch("online")
	assert.Equal(t, "crit", trigger.state.Name, "Trigger's state must be 'crit'")

	trigger.Fail("<test value>")
	assert.Equal(t, "crit", trigger.state.Name, "Trigger's state must be 'crit'")

	trigger.Touch("online")
	assert.Equal(t, "crit", trigger.state.Name, "Trigger's state must be 'crit'")

	trigger.Touch("online")
	assert.Equal(t, "good", trigger.state.Name, "Trigger's state must be 'good'")
	assert.Equal(t, 2, callCnt, "callback must be trigger only twice (fail + good)")
}
