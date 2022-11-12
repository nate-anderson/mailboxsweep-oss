package mailsweeper

var _ Inbox = &SMTPInbox{}
var _ Folder = &SMTPFolder{}
var _ Provider = &SMTPProvider{}
