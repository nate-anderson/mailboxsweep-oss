package mailboxsweep

type MessageID interface {
	uint32 | string
}

type ChangeSet struct {
	Inbox  string `dynamodbav:"inbox"`
	Folder string `dynamodbav:"folder"`
	// DeletionSet must be a set of specific message IDs (UIDs for IMAP). This is the only way to identify
	// messages across folders and sessions.
	UIDs []string `dynamodbav:"uids"`
}

func (d *ChangeSet) AddUID(uid string) {
	d.UIDs = append(d.UIDs, uid)
}
