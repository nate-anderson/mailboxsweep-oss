package classification

import (
	"context"
	"fmt"
	"mailboxsweep/changeset"
	"mailboxsweep/inbox"
	"mailboxsweep/pkg/syncmap"
)

// A message must have exactly one value for every dimension it is classified by
// example: { "sender": "nate@gmail.com" }, { "sender_domain" : "costco.com" }

type Organizer[messageIDType changeset.MessageID] struct {
	inbox       inbox.Inbox[messageIDType]
	repo        ClassificationRepository
	folders     []inbox.Folder[messageIDType]
	classifiers []Classifier
}

func NewOrganizer[messageIDType changeset.MessageID](inbox inbox.Inbox[messageIDType], repo ClassificationRepository, folders []inbox.Folder[messageIDType], classifiers []Classifier) *Organizer[messageIDType] {
	return &Organizer[messageIDType]{
		inbox:       inbox,
		repo:        repo,
		folders:     folders,
		classifiers: classifiers,
	}
}

func (o Organizer[messageIDType]) Organize(ctx context.Context) error {
	/*
		This function should accept a chan Message and process each message as it is read,
		traversing the tree and calling the classifier functions one at a time.
		This reduces the cost of classifying all the messages because it will be done
		concurrently with fetching them.
	*/

	root := &SafeClassification{
		Label:    "All Messages",
		Children: syncmap.New[string, *SafeClassification](),
	}

	validityKeys, err := o.inbox.ValidityKeys()
	if err != nil {
		return fmt.Errorf("failed to retrieve inbox UidValidities: %w", err)
	}
	selectedFolderValidityKeys := map[string]uint32{}

	m := make(chan inbox.Message)
	e := make(chan error)
	for _, f := range o.folders {
		name := f.Name()
		selectedFolderValidityKeys[name] = validityKeys[name]

		go func() {
			for msg := range m {
				go o.organizeMessage(msg, root)
			}
		}()

		msgCount := f.Count()
		go f.Messages(1, msgCount, m, e)
	}

	err = <-e
	if err != nil {
		return err
	}

	rootDTO := root.Classification()
	err = o.repo.Store(ctx, o.inbox.Name(), &rootDTO, selectedFolderValidityKeys)
	return err
}

func (o Organizer[messageIDType]) organizeMessage(message inbox.Message, root *SafeClassification) {
	node := root
	path := []string{}
	for _, classifier := range o.classifiers {
		label := classifier.Classify(message)
		key := classifier.Key(message)
		dimension := classifier.Dimension()
		child, ok := node.Children.Get(key)
		if !ok {
			child = NewSafeClassification(dimension, label)
		} else {
		}
		node.Children.Set(key, child)
		path = append(path, label)
		node = child
		node.Count++
	}
	node.Messages = append(node.Messages, message)
	// only append to leaf node
	root.Count++
}
