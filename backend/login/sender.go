package login

import (
	"context"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

type SendRequest struct {
	Expires   time.Time
	Token     string
	Signature string
	Address   string
	// we may not have this, so it's optional
	Name *string
	// @TODO unimplemented
	IsFirstLogin bool
}

// Sender sends a login link to a user
type Sender interface {
	Send(ctx context.Context, req SendRequest) error
}

type LogSender struct {
	log     *logrus.Logger
	baseURL string
}

func (l LogSender) Send(ctx context.Context, req SendRequest) error {
	query := url.Values{}
	query.Set("token", req.Token)
	query.Set("sig", req.Signature)
	u, err := url.Parse(l.baseURL)
	if err != nil {
		return err
	}
	u.RawQuery = query.Encode()
	l.log.Infof("LOGIN LINK for %s: %s (expires %s)", req.Address, u.String(), req.Expires.Format(time.Kitchen))
	return nil
}

func NewLogSender(log *logrus.Logger, baseURL string) *LogSender {
	return &LogSender{
		log,
		baseURL,
	}
}

type testSender struct {
	req SendRequest
	err error
}

func (t *testSender) Send(ctx context.Context, req SendRequest) error {
	t.req = req
	return t.err
}
