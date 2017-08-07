package main

import "testing"

func TestStateMatrixEq(t *testing.T) {
    matrix := map[interface{}]map[interface{}]bool{
        "123"        : {"123": true , "12": false, 123: false},
        float32(123) : {"123": false, "12": false, 123: true, float32(123.0): true},
    }
    for svalue, tests := range matrix {
        for tvalue, result := range tests {
            s := State{
                counterMax: 1,
                value: svalue,
            }
            s.Touch(tvalue)
            if s.IsActive() != result {
                t.Errorf("Wrong state %#v for comparision %s == %s", s.IsActive(), svalue, tvalue)
            }
        }
    }
}

func TestStateMatrixLt(t *testing.T) {
    matrix := map[interface{}]map[interface{}]bool{
        float32(123) : {"123": false, 122: true, 124: false, float32(122.0): true, float32(124.0): false},
        float32(0) : {"123": false, 122: false, 124: false, float32(122.0): false, float32(124.0): false},
    }
    for svalue, tests := range matrix {
        for tvalue, result := range tests {
            s := State{
                counterMax: 1,
                value: svalue,
                operator: "<",
            }
            s.Touch(tvalue)
            if s.IsActive() != result {
                t.Errorf("Wrong state %#v for comparision %s < %s", s.IsActive(), svalue, tvalue)
            }
        }
    }
}

func TestStateMatrixGt(t *testing.T) {
    matrix := map[interface{}]map[interface{}]bool{
        float32(123) : {"123": false, 122: false, 124: true, float32(122.0): false, float32(124.0): true},
        float32(0) : {"123": false, 122: true, 124: true, float32(122.0): true, float32(124.0): true},
    }
    for svalue, tests := range matrix {
        for tvalue, result := range tests {
            s := State{
                counterMax: 1,
                value: svalue,
                operator: ">",
            }
            s.Touch(tvalue)
            if s.IsActive() != result {
                t.Errorf("Wrong state %#v for comparision %s > %s", s.IsActive(), svalue, tvalue)
            }
        }
    }
}

func TestStateMatrixLte(t *testing.T) {
    matrix := map[interface{}]map[interface{}]bool{
        float32(123) : {"123": false, 122: true, 124: false, float32(122.0): true, float32(124.0): false, 123: true, float32(123): true},
        float32(0) : {"123": false, 122: false, 124: false, float32(122.0): false, float32(124.0): false, 0: true, float32(0): true},
    }
    for svalue, tests := range matrix {
        for tvalue, result := range tests {
            s := State{
                counterMax: 1,
                value: svalue,
                operator: "<=",
            }
            s.Touch(tvalue)
            if s.IsActive() != result {
                t.Errorf("Wrong state %#v for comparision %s <= %s", s.IsActive(), svalue, tvalue)
            }
        }
    }
}

func TestStateMatrixGte(t *testing.T) {
    matrix := map[interface{}]map[interface{}]bool{
        float32(123) : {"123": false, 122: false, 124: true, float32(122.0): false, float32(124.0): true, 123: true, float32(123): true},
        float32(0) : {"123": false, 122: true, 124: true, float32(122.0): true, float32(124.0): true, 0: true, float32(0): true},
    }
    for svalue, tests := range matrix {
        for tvalue, result := range tests {
            s := State{
                counterMax: 1,
                value: svalue,
                operator: ">=",
            }
            s.Touch(tvalue)
            if s.IsActive() != result {
                t.Errorf("Wrong state %#v for comparision %s >= %s", s.IsActive(), svalue, tvalue)
            }
        }
    }
}

func TestStateCounter(t *testing.T) {
    s := State{
        counterMax: 3,
        value: float32(123),
        operator: "<",
    }
    s.Touch(0)
    if s.IsActive() { t.Error("Must be inactive") }

    s.Touch(123)
    if s.IsActive() { t.Error("Must be inactive") }

    s.Touch(0)
    if s.IsActive() { t.Error("Must be inactive") }

    s.Touch(0)
    if s.IsActive() { t.Error("Must be inactive") }

    s.Touch(0)
    if !s.IsActive() { t.Error("Must be active") }

    s.Touch(123)
    if s.IsActive() { t.Error("Must be inactive") }

    s.Touch(0)
    if s.IsActive() { t.Error("Must be inactive") }
}
