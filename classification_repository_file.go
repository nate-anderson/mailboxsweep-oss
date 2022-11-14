package mailsweeper

import (
	"encoding/json"
	"fmt"
	"os"
)

type FileClassificationRepository struct {
	Inbox          string          `json:"inbox"`
	Classification *Classification `json:"classification"`
}

func NewFileClassificationRepository(inbox string) *FileClassificationRepository {
	return &FileClassificationRepository{
		Inbox: inbox,
	}
}

func (r *FileClassificationRepository) Store(inbox string, classification *Classification) error {
	if inbox != r.Inbox {
		return fmt.Errorf("inbox changed during repository lifecycle (store): had' %s' but '%s' was requested", r.Inbox, inbox)
	}
	r.Classification = classification

	return nil
}

func (r *FileClassificationRepository) Fetch(inbox string) (*Classification, error) {
	if inbox != r.Inbox {
		return &Classification{}, fmt.Errorf("inbox changed during repository lifecycle (fetch): had' %s' but '%s' was requested", r.Inbox, inbox)
	}

	return r.Classification, fmt.Errorf("classification for inbox/folder %s not found", inbox)
}

func (r FileClassificationRepository) Write(fileName string) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	return os.WriteFile(fileName, data, 0666)
}
