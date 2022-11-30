package mailboxsweep

import (
	"errors"
	"strings"
)

type MessageID interface {
	uint32 | string
}

type UpdateOperation string

const (
	MarkRead UpdateOperation = "MARK_READ"
	Delete   UpdateOperation = "DELETE"
)

var allUpdateOperations = []UpdateOperation{MarkRead, Delete}

type ChangeSet struct {
	Inbox  string `dynamodbav:"inbox"`
	Folder string `dynamodbav:"folder"`
	// is this a "mark-read" selection, or a "delete" selection?
	Operation UpdateOperation `dynamodbav:"update_operation"`
	// DeletionSet must be a set of specific message IDs (UIDs for IMAP). This is the only way to identify
	// messages across folders and sessions.
	UIDs []string `dynamodbav:"uids"`
}

func (d *ChangeSet) AddUID(uid string) {
	d.UIDs = append(d.UIDs, uid)
}

var (
	ErrNoOperation       = errors.New("change set requires an operation")
	ErrInvalidOperation  = errors.New("invalid operation in change set")
	ErrNoUIDSelection    = errors.New("no UIDs selected for change set")
	ErrNoInboxSelection  = errors.New("no inbox selected for change set")
	ErrNoFolderSelection = errors.New("no folder selected for change set")
)

func (d ChangeSet) Validate() error {
	if d.Operation == "" {
		return ErrNoOperation
	}

	if !contains(allUpdateOperations, d.Operation) {
		return ErrInvalidOperation
	}

	if len(d.UIDs) == 0 {
		return ErrNoUIDSelection
	}

	if strings.TrimSpace(d.Inbox) == "" {
		return ErrNoInboxSelection
	}

	if strings.TrimSpace(d.Folder) == "" {
		return ErrNoFolderSelection
	}

	return nil
}
