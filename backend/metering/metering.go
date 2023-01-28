package metering

/*
Metering implements metered connections
Required to be a good IMAP citizen
*/

import (
	"errors"
	"fmt"
	"net"
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
	// used for persisting quota usage
	updateChan chan Update
}

func newConn(conn net.Conn, quota int, updateChan chan Update) *Conn {
	return &Conn{
		Conn:            conn,
		bytesTransfered: 0,
		quota:           quota,
		updateChan:      updateChan,
	}
}

var ErrQuotaReached = errors.New("metering connection: quota reached")

func (c *Conn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if err != nil {
		return n, err
	}

	c.bytesTransfered += n

	if c.updateChan != nil {
		update := Update{
			TotalBytes: c.bytesTransfered,
		}
		c.updateChan <- update
	}

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

	c.bytesTransfered += n

	if c.updateChan != nil {
		update := Update{
			TotalBytes: c.bytesTransfered,
		}
		c.updateChan <- update
	}

	if c.quota == 0 {
		return n, err
	}

	if c.bytesTransfered >= c.quota {
		return n, fmt.Errorf("%d bytes: %s", c.quota, ErrQuotaReached)
	}

	return n, err
}

type Dialer struct {
	quota      int
	updateChan chan Update
}

type DialerOpt func(d *Dialer)

// WithQuota specifies the quota (read + write combined) for the connection, in bytes
func WithQuota(quota int) func(d *Dialer) {
	return func(d *Dialer) {
		d.quota = quota
	}
}

func NewDialer(updateChan chan Update, opts ...DialerOpt) *Dialer {
	d := &Dialer{
		updateChan: updateChan,
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

	return newConn(conn, d.quota, d.updateChan), nil
}
