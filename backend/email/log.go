package email

import (
	"context"

	"github.com/sirupsen/logrus"
)

type LogSender struct {
	log *logrus.Logger
}

func (s LogSender) Send(ctx context.Context, address, template string, data map[string]string) error {
	s.log.
		WithField("address", address).
		WithField("template", template).
		WithField("data", data).
		Info("Email Sent!")

	return nil
}

func NewLogSender(log *logrus.Logger) *LogSender {
	return &LogSender{log}
}
