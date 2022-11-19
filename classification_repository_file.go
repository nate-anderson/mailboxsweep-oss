package mailsweeper

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type FileClassificationRepository struct {
	dir string
}

/*
This is really for local testing and is very I/O inefficient
*/

func NewFileClassificationRepository(dir string) *FileClassificationRepository {
	return &FileClassificationRepository{
		dir: dir,
	}
}

func (r FileClassificationRepository) formatFileName(inbox string) string {
	return fmt.Sprintf("classification_%s.json", inbox)
}

func (r FileClassificationRepository) formatFilePath(inbox string) string {
	return filepath.Join(r.dir, r.formatFileName(inbox))
}

func (r FileClassificationRepository) Store(ctx context.Context, inbox string, classification *Classification) error {
	filename := r.formatFilePath(inbox)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	return encoder.Encode(classification)
}

func (r FileClassificationRepository) FetchSummary(ctx context.Context, inbox string) (*Classification, error) {
	file, err := os.Open(
		r.formatFilePath(inbox),
	)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var classification Classification
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&classification); err != nil {
		return nil, err
	}

	summary, err := classification.withoutMessages(), nil
	return &summary, err
}

func (r FileClassificationRepository) FetchLeaf(ctx context.Context, inbox string, path []string) (*Classification, error) {
	file, err := os.Open(
		r.formatFilePath(inbox),
	)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var classification Classification
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&classification); err != nil {
		return nil, err
	}

	cl, ok := classification.Traverse(path)
	if !ok {
		return nil, fmt.Errorf("could not find classification at path %v in tree", path)
	}
	return &cl, nil
}
