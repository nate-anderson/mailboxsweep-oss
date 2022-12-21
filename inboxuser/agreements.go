package inboxuser

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed agreements/*
var agreementsFS embed.FS

// embedded documents from /agreements
type AgreementDocument struct {
	Code    Agreement `yaml:"-"`
	Name    string    `yaml:"name"`
	Updated string    `yaml:"updated"`
	Body    string    `yaml:"body"`
}

const dir = "agreements"
const ext = ".yml"

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
		if err = yaml.NewDecoder(file).Decode(&doc); err != nil {
			return nil, err
		}

		key := strings.TrimSuffix(entry.Name(), ext)
		agr, err := ParseAgreement(key)
		if err != nil {
			return nil, fmt.Errorf("agreements directory contained file with unknown agreement code: %w", err)
		}

		docs[agr] = doc
	}

	return docs, nil
}
