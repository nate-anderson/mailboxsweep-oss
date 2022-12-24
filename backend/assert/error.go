package assert

import (
	"errors"
	"fmt"
	"runtime"
	"testing"
)

type errorAssertion struct {
	t      *testing.T
	actual error
	reason string
}

func ThatError(t *testing.T, actual error) *errorAssertion {
	return &errorAssertion{
		t:      t,
		actual: actual,
	}
}

func (a *errorAssertion) SoThat(reason string) *errorAssertion {
	a.reason = reason
	return a
}

func (a errorAssertion) failf(msg string, args ...any) {
	_, file, line, ok := runtime.Caller(2)
	m := fmt.Sprintf(msg, args...)
	if a.reason != "" {
		m += fmt.Sprintf(" (%s)", a.reason)
	}
	if !ok {
		a.t.Error(m)
	}
	a.t.Errorf("%s in %s:%d", m, file, line)
}

func (a *errorAssertion) IsNil() *errorAssertion {
	if a.actual != nil {
		a.failf("Expected error to be nil: %v", a.actual)
		a.t.FailNow()
	}
	return a
}

func (a *errorAssertion) IsNotNil() *errorAssertion {
	if a.actual == nil {
		a.failf("Expected error to be non-nil: %v", a.actual)
		a.t.FailNow()
	}
	return a
}

func (a *errorAssertion) Is(parent error) *errorAssertion {
	if !errors.Is(a.actual, parent) {
		a.failf("Expected error %s to have parent %s: it did not", a.actual, parent)
		a.t.FailNow()
	}
	return a
}
