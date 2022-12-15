package inboxtesting

import (
	"fmt"
	"mailboxsweep"
	"mailboxsweep/inbox"
)

type TestInbox struct {
	name         string
	validityKeys map[string]uint32

	messages map[string][]inbox.Message

	foldersError      error
	folderError       error
	validityKeysError error
}

func (t TestInbox) Name() string {
	return t.name
}

func (t TestInbox) Folders() ([]string, error) {
	if t.foldersError != nil {
		return []string{}, t.foldersError
	}
	folders := make([]string, 0, len(t.messages))
	for key := range t.messages {
		folders = append(folders, key)
	}
	return folders, nil
}

func (t TestInbox) Folder(name string) (inbox.Folder, error) {
	if t.folderError != nil {
		return nil, t.folderError
	}

	messages, ok := t.messages[name]
	if !ok {
		return nil, fmt.Errorf("folder '%s' not found: %w", name, mailboxsweep.ErrNotFound)
	}

	return &TestFolder{
		FolderName:     name,
		FolderMessages: messages,
	}, nil
}

func (t TestInbox) ValidityKeys() (map[string]uint32, error) {
	if t.validityKeysError != nil {
		return nil, t.validityKeysError
	}

	return t.validityKeys, nil
}
