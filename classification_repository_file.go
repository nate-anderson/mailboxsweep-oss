package mailsweeper

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type FileClassificationRepository struct{}

func NewFileClassificationRepository() *FileClassificationRepository {
	return &FileClassificationRepository{}
}

func (r *FileClassificationRepository) formatFileName(inbox string) string {
	return fmt.Sprintf("classification_%s.json", inbox)
}

func (r *FileClassificationRepository) Store(ctx context.Context, inbox string, classification *Classification) error {
	filename := r.formatFileName(inbox)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	return encoder.Encode(classification)
}

func (r *FileClassificationRepository) Fetch(ctx context.Context, inbox string, path []string, includeMessages bool) (*Classification, error) {
	file, err := os.Open(
		r.formatFileName(inbox),
	)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var classification *Classification
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&classification); err != nil {
		return nil, err
	}

	if len(path) == 0 {
		return classification, nil
	}

	node := classification
	for _, entry := range path {
		var ok bool
		node, ok = node.Children.Get(entry)
		if !ok {
			return nil, fmt.Errorf("no classification found at path %s", strings.Join(path, "/"))
		}
	}

	if !includeMessages {
		stripMessages(node)
	}
	return node, nil
}
