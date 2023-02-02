package metering

/*
Metering implements metered connections
Required to be a good IMAP citizen

Note that this package has context.Context struct fields
This is necessary because io.Reader and io.Writer do not
take context arguments
*/

import (
	"context"
	"errors"
	"fmt"
	"mailboxsweep"
	"math"
	"net"
	"time"
)

const (
	Kilobyte = 1000
	Megabyte = Kilobyte * 1000
)

type Update struct {
	TotalBytes int
}

type Conn struct {
	net.Conn
	bytesTransfered int
	quota           int

	updateInterval int
	lastUpdate     int

	address string
	ctx     context.Context

	repo UsageRepository
}

func newConn(ctx context.Context, conn net.Conn, address string, repo UsageRepository, quota int) *Conn {
	return &Conn{
		ctx:             ctx,
		Conn:            conn,
		bytesTransfered: 0,
		quota:           quota,
		address:         address,
		updateInterval:  Megabyte,
		repo:            repo,
	}
}

func (c *Conn) incrementBytesTransferred(n int) error {
	c.bytesTransfered += n
	if c.repo != nil && (c.bytesTransfered-c.lastUpdate >= c.updateInterval) {
		// attempt to persist state
		c.lastUpdate = c.bytesTransfered
		return retryLinear(3, 3, func() error {
			return c.repo.Update(c.ctx, c.address, c.bytesTransfered)
		})
	}
	return nil
}

var ErrQuotaReached = errors.New("metering connection: quota reached")

func (c *Conn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if err != nil {
		return n, err
	}

	go c.incrementBytesTransferred(n)

	if c.quota == 0 {
		return n, err
	}

	if c.bytesTransfered >= c.quota {
		return n, fmt.Errorf("%d bytes: %w", c.quota, ErrQuotaReached)
	}

	return n, err
}

func (c *Conn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if err != nil {
		return n, err
	}

	go c.incrementBytesTransferred(n)

	if c.quota == 0 {
		return n, err
	}

	if c.bytesTransfered >= c.quota {
		return n, fmt.Errorf("%d bytes: %s", c.quota, ErrQuotaReached)
	}

	return n, err
}

func (c *Conn) Close() error {
	return c.Conn.Close()
}

type Dialer struct {
	ctx     context.Context
	quota   int
	address string
	repo    UsageRepository
}

type DialerOpt func(d *Dialer)

// WithQuota specifies the quota (read + write combined) for the connection, in bytes
func WithQuota(quota int) func(d *Dialer) {
	return func(d *Dialer) {
		d.quota = quota
	}
}

func NewDialer(ctx context.Context, address string, repo UsageRepository, opts ...DialerOpt) *Dialer {
	d := &Dialer{
		ctx:     ctx,
		address: address,
		repo:    repo,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *Dialer) Dial(network, addr string) (net.Conn, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}

	wrapped := newConn(d.ctx, conn, d.address, d.repo, d.quota)

	if d.repo != nil {
		existingBytes, err := d.repo.CheckUsage(d.ctx, d.address)
		if err == nil {
			wrapped.bytesTransfered = existingBytes.BytesUsed
		} else if errors.Is(err, mailboxsweep.ErrNotFound) {
			return wrapped, nil
		} else if err != nil {
			return nil, err
		}
	}

	return wrapped, nil
}

// retry with linear backoff
func retryLinear(numRetries int, baseDurationSeconds int, fn func() error) error {
	var err error
	for i := 1; i <= numRetries; i++ {
		backoff := int(math.Pow(float64(baseDurationSeconds), float64(i)))
		time.Sleep(time.Duration(backoff))
		if err = fn(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("failure after %d retries: %w", numRetries, err)
}
