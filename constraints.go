package mailboxsweep

var (
	_ Inbox[uint32]    = &IMAPInbox{}
	_ Folder[uint32]   = &IMAPFolder{}
	_ Provider[uint32] = &IMAPProvider{}
)

var (
	_ Classifier = SenderDomainClassifier{}
	_ Classifier = SenderUsernameClassifier{}
	_ Classifier = SendDateClassifier{}
)

var (
	_ ClassificationRepository = &FileClassificationRepository{}
)
