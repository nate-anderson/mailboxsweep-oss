package mailsweeper

import (
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

func (r *FileClassificationRepository) Store(inbox string, classification *Classification) error {
	filename := r.formatFileName(inbox)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	return encoder.Encode(classification)
}

func (r *FileClassificationRepository) Fetch(inbox string, path []string) (*Classification, error) {
	file, err := os.Open(
		r.formatFileName(inbox),
	)
	if err != nil {
		return nil, err
	}

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
	return node, nil
}

func (r FileClassificationRepository) write(fileName string) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	return os.WriteFile(fileName, data, 0666)
}
