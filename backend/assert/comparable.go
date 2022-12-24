package assert

import (
	"fmt"
	"runtime"
	"testing"
)

type assertion[E comparable] struct {
	t      *testing.T
	actual E
	reason string
}

func That[E comparable](t *testing.T, actual E) *assertion[E] {
	return &assertion[E]{
		t:      t,
		actual: actual,
	}
}

func (a *assertion[E]) SoThat(reason string) *assertion[E] {
	a.reason = reason
	return a
}

func (a assertion[E]) failf(msg string, args ...any) {
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

func (a *assertion[E]) Equals(expected E) *assertion[E] {
	if a.actual != expected {
		a.failf("Value %+v did not match expected %+v", a.actual, expected)
	}
	return a
}

func (a *assertion[E]) NotEqual(expected E) *assertion[E] {
	if a.actual == expected {
		a.failf("Value %+v should not equal %+v", a.actual, expected)
	}
	return a
}
