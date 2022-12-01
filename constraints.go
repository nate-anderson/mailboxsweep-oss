package mailboxsweep

var (
	_ Inbox[uint32]    = &IMAPInbox{}
	_ Folder[uint32]   = &IMAPFolder{}
	_ Provider[uint32] = &IMAPProvider{}
)
