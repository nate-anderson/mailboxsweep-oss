package inboxtesting

import (
	"context"
	"mailboxsweep/inbox"
)

type TestFolder struct {
	FolderName     string
	FolderMessages []inbox.Message

	MessagesError error
	DeleteError   error
	MarkReadError error

	MessagesStart uint32
	MessagesEnd   uint32

	Deletes   []string
	MarkReads []string
}

func (t TestFolder) Name() string {
	return t.FolderName
}

func (t *TestFolder) Messages(_ context.Context, start, end uint32, m chan<- inbox.Message, e chan<- error) {
	t.MessagesStart = start
	t.MessagesEnd = end

	if t.MessagesError != nil {
		e <- t.MessagesError
	}

	for _, msg := range t.FolderMessages {
		m <- msg
	}

	e <- nil
}

func (t TestFolder) Count() uint32 {
	return uint32(len(t.FolderMessages))
}

func (t TestFolder) UnreadCount() uint32 {
	var count uint32
	for _, m := range t.FolderMessages {
		if !m.Read {
			count++
		}
	}
	return count
}

func (t *TestFolder) Delete(uids []string) error {
	if t.DeleteError != nil {
		return t.DeleteError
	}

	t.Deletes = append(t.Deletes, uids...)
	return nil
}

func (t *TestFolder) MarkRead(uids []string) error {
	if t.MarkReadError != nil {
		return t.MarkReadError
	}

	t.MarkReads = append(t.MarkReads, uids...)
	return nil
}
