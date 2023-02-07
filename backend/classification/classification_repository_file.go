package classification

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type FileClassificationRepository struct {
	dir      string
	pageSize int
}

/*
This is really for local testing and is very I/O inefficient
*/

func NewFileClassificationRepository(dir string, pageSize int) *FileClassificationRepository {
	return &FileClassificationRepository{
		dir:      dir,
		pageSize: pageSize,
	}
}

func (r FileClassificationRepository) formatFileName(inbox string) string {
	return fmt.Sprintf("classification_%s.json", inbox)
}

func (r FileClassificationRepository) formatFilePath(inbox string) string {
	return filepath.Join(r.dir, r.formatFileName(inbox))
}

func (r FileClassificationRepository) Store(ctx context.Context, inbox string, classification *Classification, validityKeys map[string]uint32) error {
	filename := r.formatFilePath(inbox)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	return encoder.Encode(ClassificationDocument{
		Meta: ClassificationMeta{
			Created:      time.Now(),
			ValidityKeys: validityKeys,
		},
		Classification: classification,
	})
}

func (r FileClassificationRepository) FetchSummary(ctx context.Context, inbox string, page int) (*ClassificationDocument, error) {
	file, err := os.Open(
		r.formatFilePath(inbox),
	)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var classification ClassificationDocument
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&classification); err != nil {
		return nil, err
	}

	cl := classification.Classification.withoutMessages()

	pageStart := (page * r.pageSize) + 1
	pageEnd := r.pageSize * (page + 1)

	for key, value := range cl.Children {
		if value.RankIndex == nil || *value.RankIndex < pageStart || *value.RankIndex > pageEnd {
			delete(cl.Children, key)
		}
	}
	classification.Classification = &cl
	return &classification, err
}

func (r FileClassificationRepository) FetchLeaf(ctx context.Context, inbox string, path []string) (*Classification, error) {
	file, err := os.Open(
		r.formatFilePath(inbox),
	)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var classificationDoc ClassificationDocument
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&classificationDoc); err != nil {
		return nil, err
	}

	cl, ok := classificationDoc.Classification.Traverse(path)
	if !ok {
		return nil, fmt.Errorf("could not find classification at path %v in tree", path)
	}
	return &cl, nil
}
