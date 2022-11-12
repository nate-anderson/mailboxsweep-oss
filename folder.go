package mailsweeper

type Folder interface {
	// @TODO don't use IMAP type here
	Messages(start, end uint32) ([]Message, error)
	Count() uint32
}
