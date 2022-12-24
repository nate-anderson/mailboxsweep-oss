package assert

import (
	"fmt"
	"runtime"
	"testing"

	"golang.org/x/exp/constraints"
)

type orderedAssertion[E constraints.Ordered] struct {
	t      *testing.T
	actual E
	reason string
}

func ThatOrdered[E constraints.Ordered](t *testing.T, actual E) *orderedAssertion[E] {
	return &orderedAssertion[E]{
		t:      t,
		actual: actual,
	}
}

func (a *orderedAssertion[E]) SoThat(reason string) *orderedAssertion[E] {
	a.reason = reason
	return a
}

func (a orderedAssertion[E]) failf(msg string, args ...any) {
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

func (a *orderedAssertion[E]) Equals(expected E) *orderedAssertion[E] {
	if a.actual != expected {
		a.failf("Value %+v did not match expected %+v", a.actual, expected)
	}
	return a
}

func (a *orderedAssertion[E]) GreaterThan(expected E) *orderedAssertion[E] {
	if a.actual <= expected {
		a.failf("Value %+v should be greater than %+v", a.actual, expected)
	}
	return a
}

func (a *orderedAssertion[E]) LessThan(expected E) *orderedAssertion[E] {
	if a.actual >= expected {
		a.failf("Value %+v should be less than %+v", a.actual, expected)
	}
	return a
}

func (a *orderedAssertion[E]) GreaterThanOrEqual(expected E) *orderedAssertion[E] {
	if a.actual < expected {
		a.failf("Value %+v should be greater than or equal to %+v", a.actual, expected)
	}
	return a
}

func (a *orderedAssertion[E]) LessThanOrEqual(expected E) *orderedAssertion[E] {
	if a.actual > expected {
		a.failf("Value %+v should be less than or equal to %+v", a.actual, expected)
	}
	return a
}
