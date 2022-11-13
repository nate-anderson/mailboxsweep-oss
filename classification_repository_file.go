package mailsweeper

import "fmt"

type FileClassificationRepository struct {
	inbox                 string                    `json:"inbox"`
	folderClassifications map[string]Classification `json:"inbox"`
}

func (r *FileClassificationRepository) Store(inbox, folder string, classification Classification) error {
	if inbox != r.inbox {
		return fmt.Errorf("inbox changed during repository lifecycle (store): had' %s' but '%s' was requested", r.inbox, inbox)
	}
	r.folderClassifications[folder] = classification

	return nil
}

func (r *FileClassificationRepository) Fetch(inbox, folder string) (Classification, error) {
	if inbox != r.inbox {
		return Classification{}, fmt.Errorf("inbox changed during repository lifecycle (fetch): had' %s' but '%s' was requested", r.inbox, inbox)
	}

	cl, ok := r.folderClassifications[folder]
	if ok {
		return cl, nil
	}

	return Classification{}, fmt.Errorf("classification for inbox/folder %s/%s not found", inbox, folder)
}
