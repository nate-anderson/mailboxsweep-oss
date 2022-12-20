package inboxuser

import (
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

//go:embed agreements/*
var agreementsFS embed.FS

// embedded documents from /agreements
type AgreementDocument struct {
	Code    Agreement `json:"-"`
	Name    string    `json:"name"`
	Updated string    `json:"updated"`
	Body    string    `json:"body"`
}

const dir = "agreements"

func readAgreements() (map[Agreement]AgreementDocument, error) {
	dirEntries, err := agreementsFS.ReadDir("agreements")
	if err != nil {
		return nil, err
	}

	docs := make(map[Agreement]AgreementDocument, len(dirEntries))
	for _, entry := range dirEntries {
		path := filepath.Join(dir, entry.Name())
		file, err := agreementsFS.Open(path)
		if err != nil {
			return nil, err
		}

		doc := AgreementDocument{}
		if err = json.NewDecoder(file).Decode(&doc); err != nil {
			return nil, err
		}

		key := strings.TrimSuffix(entry.Name(), ".json")
		agr, err := ParseAgreement(key)
		if err != nil {
			return nil, fmt.Errorf("agreements directory contained file with unknown agreement code: %w", err)
		}

		docs[agr] = doc
	}

	return docs, nil
}
