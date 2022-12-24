package assert

import (
	"fmt"
	"runtime"
	"testing"
)

type mapAssertion[keyE, valueE comparable] struct {
	t      *testing.T
	actual map[keyE]valueE
	reason string
}

func ThatMap[keyE, valueE comparable](t *testing.T, actual map[keyE]valueE) *mapAssertion[keyE, valueE] {
	return &mapAssertion[keyE, valueE]{
		t:      t,
		actual: actual,
	}
}

func (a *mapAssertion[keyE, valueE]) SoThat(reason string) *mapAssertion[keyE, valueE] {
	a.reason = reason
	return a
}

func (a mapAssertion[keyE, valueE]) failf(msg string, args ...any) {
	_, file, line, ok := runtime.Caller(2)
	m := fmt.Sprintf(msg, args...)
	if a.reason != "" {
		m += fmt.Sprintf(" (%s)", a.reason)
	}
	if !ok {
		a.t.Errorf(m, args...)
	}
	a.t.Errorf("%s in %s:%d", m, file, line)
}

func (a *mapAssertion[keyE, valueE]) ContainsKey(key keyE) *mapAssertion[keyE, valueE] {
	_, ok := a.actual[key]
	if !ok {
		a.failf("Expected map to contain key %s, it did not", key)
	}
	return a
}

func (a *mapAssertion[keyE, valueE]) HasLength(n int) *mapAssertion[keyE, valueE] {
	if len(a.actual) != n {
		a.failf("Map did not have expected length %d (map %v)", n, a.actual)
	}
	return a
}

func (a *mapAssertion[keyE, valueE]) HasValueAt(key keyE, expected valueE) *mapAssertion[keyE, valueE] {
	val, ok := a.actual[key]
	if !ok {
		a.failf("Expected map to have key %v with value %v: it did not", key, expected)
	}
	if val != expected {
		a.failf("Expected map value at key %v to equal %v: got %v", key, expected, val)
	}
	return a
}
