package assert

import (
	"fmt"
	"runtime"
	"testing"
)

type sliceAssertion[E comparable] struct {
	t      *testing.T
	actual []E
	reason string
}

func ThatSlice[E comparable](t *testing.T, actual []E) *sliceAssertion[E] {
	return &sliceAssertion[E]{
		t:      t,
		actual: actual,
	}
}

func (a *sliceAssertion[E]) SoThat(reason string) *sliceAssertion[E] {
	a.reason = reason
	return a
}

func (a sliceAssertion[E]) failf(msg string, args ...any) {
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

func (a *sliceAssertion[E]) Contains(member E) *sliceAssertion[E] {
	for _, m := range a.actual {
		if m == member {
			return a
		}
	}
	a.failf("Expected slice to contain %v, but it did not (slice %+v)", member, a.actual)
	return a
}

func (a *sliceAssertion[E]) NotContains(member E) *sliceAssertion[E] {
	for _, m := range a.actual {
		if m == member {
			a.failf("Expected slice not to contain %v, but it did (slice %+v)", member, a.actual)
		}
	}
	return a
}

func (a *sliceAssertion[E]) HasLength(n int) *sliceAssertion[E] {
	if len(a.actual) != n {
		a.failf("Slice did not have expected length %d (got %d, slice %+v)", n, len(a.actual), a.actual)
	}
	return a
}

func (a *sliceAssertion[E]) Equals(expected []E) *sliceAssertion[E] {
	if len(a.actual) != len(expected) {
		a.failf("Slices not equal:\nActual: %v\nExpected: %v", a.actual, expected)
		return a
	}
	for i, e := range expected {
		if e != a.actual[i] {
			a.failf("Slices not equal:\nActual: %v\nExpected: %v", a.actual, expected)
			a.t.FailNow()
		}
	}
	return a
}

func (a *sliceAssertion[E]) ContainsAll(expected []E) *sliceAssertion[E] {
	if len(a.actual) != len(expected) {
		a.failf("Actual does not contain all of expected:\nExpected: %+v\nActual: %+v", expected, a.actual)
		return a
	}
	for _, e := range expected {
		a.Contains(e)
	}
	return a
}
