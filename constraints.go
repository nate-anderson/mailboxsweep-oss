package mailsweeper

var (
	_ Inbox    = &SMTPInbox{}
	_ Folder   = &SMTPFolder{}
	_ Provider = &SMTPProvider{}
)

var (
	_ Classifier = SenderDomainClassifier{}
	_ Classifier = SenderUsernameClassifier{}
	_ Classifier = SendDateClassifier{}
)
