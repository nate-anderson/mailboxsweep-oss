package inboxtesting

import "mailboxsweep/inbox"

type TestInbox struct {
	name         string
	folders      []string
	validityKeys map[string]uint32

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
	return t.folders, nil
}

func (t TestInbox) Folder(name string) (inbox.Folder, error) {
	if t.folderError != nil {
		return nil, t.folderError
	}

	return &TestFolder{
		FolderName: name,
	}, nil
}

func (t TestInbox) ValidityKeys() (map[string]uint32, error) {
	if t.validityKeysError != nil {
		return nil, t.validityKeysError
	}

	return t.validityKeys, nil
}
