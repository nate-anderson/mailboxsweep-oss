package email

import "context"

type Sender interface {
	Send(ctx context.Context, address, template string, data map[string]string) error
}
