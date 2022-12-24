package changeset

/*
changeset defines user-selected sets of message identifiers that mailboxsweep will apply changes to
*/

import (
	"errors"
	"strings"

	"mailboxsweep/pkg/slice"
)

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
	Operation UpdateOperation `dynamodbav:"operation"`
	// specify a set of paths in a classified inbox
	Paths []string `dynamodbav:"paths"`
}

func (d *ChangeSet) AddPath(path string) {
	d.Paths = append(d.Paths, path)
}

var (
	ErrNoOperation       = errors.New("change set requires an operation")
	ErrInvalidOperation  = errors.New("invalid operation in change set")
	ErrNoPathSelection   = errors.New("no paths selected for change set")
	ErrNoInboxSelection  = errors.New("no inbox selected for change set")
	ErrNoFolderSelection = errors.New("no folder selected for change set")
)

func (d ChangeSet) Validate() error {
	if d.Operation == "" {
		return ErrNoOperation
	}

	if !slice.Contains(allUpdateOperations, d.Operation) {
		return ErrInvalidOperation
	}

	if len(d.Paths) == 0 {
		return ErrNoPathSelection
	}

	if strings.TrimSpace(d.Inbox) == "" {
		return ErrNoInboxSelection
	}

	if strings.TrimSpace(d.Folder) == "" {
		return ErrNoFolderSelection
	}

	return nil
}
