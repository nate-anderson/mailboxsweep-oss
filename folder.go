package mailsweeper

// Folder is a grouping of messages within an email inbox
// called "inboxes" in IMAP but that name is confusing
type Folder interface {
	Messages(start, end uint32) ([]Message, error)
	Count() uint32
}
