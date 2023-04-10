package inbox

import "context"

// Folder is a grouping of messages within an email inbox
// called "inboxes" in IMAP but that name is confusing
type Folder interface {
	Name() string
	Messages(ctx context.Context, start, end uint32, m chan<- Message, e chan<- error)
	Count() uint32
	UnreadCount() uint32
	Delete(uids []string) error
	MarkRead(uids []string) error
}
