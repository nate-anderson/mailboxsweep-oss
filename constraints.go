package mailsweeper

var (
	_ Inbox    = &IMAPInbox{}
	_ Folder   = &IMAPFolder{}
	_ Provider = &IMAPProvider{}
)

var (
	_ Classifier = SenderDomainClassifier{}
	_ Classifier = SenderUsernameClassifier{}
	_ Classifier = SendDateClassifier{}
)

var (
	_ ClassificationRepository = &FileClassificationRepository{}
)
