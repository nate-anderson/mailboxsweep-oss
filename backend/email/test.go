package email

import "context"

type TestSender struct {
	LastAddress  string
	LastTemplate string
	LastData     map[string]string

	err error
}

func (s *TestSender) Send(_ context.Context, address, template string, data map[string]string) error {
	s.LastAddress = address
	s.LastTemplate = template
	s.LastData = data

	return s.err
}
