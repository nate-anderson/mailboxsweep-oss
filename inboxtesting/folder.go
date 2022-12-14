package inboxtesting

import "mailboxsweep/inbox"

type TestFolder struct {
	FolderName     string
	FolderMessages []inbox.Message

	MessagesError error
	DeleteError   error

	MessagesStart uint32
	MessagesEnd   uint32

	Deletes []string
}

func (t TestFolder) Name() string {
	return t.FolderName
}

func (t *TestFolder) Messages(start, end uint32, m chan<- inbox.Message, e chan<- error) {
	t.MessagesStart = start
	t.MessagesEnd = end

	if t.MessagesError != nil {
		e <- t.MessagesError
		return
	}

	for _, msg := range t.FolderMessages {
		m <- msg
	}

	e <- nil
}

func (t TestFolder) Count() uint32 {
	return uint32(len(t.FolderMessages))
}

func (t *TestFolder) Delete(uids []string) error {
	if t.DeleteError != nil {
		return t.DeleteError
	}

	t.Deletes = append(t.Deletes, uids...)
	return nil
}
