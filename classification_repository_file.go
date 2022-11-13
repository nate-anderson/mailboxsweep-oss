package mailsweeper

import "fmt"

type FileClassificationRepository struct {
	Inbox                 string                     `json:"inbox"`
	FolderClassifications map[string]*Classification `json:"folderClassificagions"`
}

func (r *FileClassificationRepository) Store(inbox, folder string, classification *Classification) error {
	if inbox != r.Inbox {
		return fmt.Errorf("inbox changed during repository lifecycle (store): had' %s' but '%s' was requested", r.Inbox, inbox)
	}
	r.FolderClassifications[folder] = classification

	return nil
}

func (r *FileClassificationRepository) Fetch(inbox, folder string) (*Classification, error) {
	if inbox != r.Inbox {
		return &Classification{}, fmt.Errorf("inbox changed during repository lifecycle (fetch): had' %s' but '%s' was requested", r.Inbox, inbox)
	}

	cl, ok := r.FolderClassifications[folder]
	if ok {
		return cl, nil
	}

	return &Classification{}, fmt.Errorf("classification for inbox/folder %s/%s not found", inbox, folder)
}
