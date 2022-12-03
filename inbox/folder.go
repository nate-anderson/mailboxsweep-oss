package inbox

import "mailboxsweep/changeset"

// Folder is a grouping of messages within an email inbox
// called "inboxes" in IMAP but that name is confusing
type Folder interface {
	Name() string
	Messages(start, end uint32, m chan<- Message, e chan<- error)
	Count() uint32
	Delete(del changeset.ChangeSet) error
}
