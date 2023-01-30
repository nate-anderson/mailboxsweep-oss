package assert

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

type stringAssertion struct {
	t      *testing.T
	actual string
	reason string
}

func ThatString(t *testing.T, actual string) *stringAssertion {
	return &stringAssertion{
		t:      t,
		actual: actual,
	}
}

func (a *stringAssertion) SoThat(reason string) *stringAssertion {
	a.reason = reason
	return a
}

func (a stringAssertion) failf(msg string, args ...any) {
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

func (a *stringAssertion) Contains(substring string) *stringAssertion {
	if !strings.Contains(a.actual, substring) {
		a.failf("Expected string to contain %s, but it did not (string %s)", substring, a.actual)
	}
	return a
}

func (a *stringAssertion) NotContains(substring string) *stringAssertion {
	if strings.Contains(a.actual, substring) {
		a.failf("Expected string not to contain %s, but it did not (string %s)", substring, a.actual)
	}
	return a
}

func (a *stringAssertion) StartsWith(prefix string) *stringAssertion {
	if !strings.HasPrefix(a.actual, prefix) {
		a.failf("Expected string to start with %s, but it did not string %s)", prefix, a.actual)
	}
	return a
}

func (a *stringAssertion) EndsWith(suffix string) *stringAssertion {
	if !strings.HasSuffix(a.actual, suffix) {
		a.failf("Expected string to end with %s, but it did not string %s)", suffix, a.actual)
	}
	return a
}

func (a *stringAssertion) HasLength(n int) *stringAssertion {
	if len(a.actual) != n {
		a.failf("String did not have expected length %d (string %v)", n, a.actual)
	}
	return a
}
