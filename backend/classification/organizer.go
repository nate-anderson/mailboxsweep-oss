package classification

import (
	"context"
	"fmt"
	"mailboxsweep/inbox"
	"mailboxsweep/pkg/syncmap"
	"sort"
	"sync"
)

// A message must have exactly one value for every dimension it is classified by
// example: { "sender": "nate@gmail.com" }, { "sender_domain" : "costco.com" }

type Organizer struct {
	inbox       inbox.Inbox
	repo        ClassificationRepository
	folders     []string
	classifiers []Classifier
}

func NewOrganizer(inbox inbox.Inbox, repo ClassificationRepository, folders []string, classifiers []Classifier) *Organizer {
	return &Organizer{
		inbox:       inbox,
		repo:        repo,
		folders:     folders,
		classifiers: classifiers,
	}
}

func (o Organizer) Organize(ctx context.Context) (*Classification, error) {
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
		return nil, fmt.Errorf("failed to retrieve inbox UidValidities: %w", err)
	}
	selectedFolderValidityKeys := map[string]uint32{}

	m := make(chan inbox.Message)
	e := make(chan error)
	wg := sync.WaitGroup{}
	for _, name := range o.folders {
		wg.Add(1)
		f, err := o.inbox.Folder(name)
		if err != nil {
			return nil, err
		}
		selectedFolderValidityKeys[name] = validityKeys[name]

		go func() {
			for {
				select {
				case msg := <-m:
					o.organizeMessage(msg, root)

				case err = <-e:
					wg.Done()
					if err != nil {
						panic(err)
					}
					return
				}
			}
		}()

		msgCount := f.Count()
		go f.Messages(1, msgCount, m, e)
	}

	wg.Wait()
	rootDTO := root.Classification()

	// assign order index to support pagination of top-level groups
	sortedKeys := make([]string, 0, len(rootDTO.Children))
	for k := range rootDTO.Children {
		sortedKeys = append(sortedKeys, k)
	}

	sort.Slice(sortedKeys, func(i, j int) bool {
		keyI := sortedKeys[i]
		keyJ := sortedKeys[j]
		return rootDTO.Children[keyI].Count > rootDTO.Children[keyJ].Count
	})

	for i, key := range sortedKeys {
		child := rootDTO.Children[key]
		rank := i
		child.RankIndex = &rank
		rootDTO.Children[key] = child
	}

	err = o.repo.Store(ctx, o.inbox.Name(), &rootDTO, selectedFolderValidityKeys)
	return &rootDTO, err
}

func (o Organizer) organizeMessage(message inbox.Message, root *SafeClassification) {
	node := root
	path := []string{}
	for _, classifier := range o.classifiers {
		label := classifier.Classify(message)
		key := classifier.Key(message)
		dimension := classifier.Dimension()
		child, ok := node.Children.Get(key)
		if !ok {
			child = NewSafeClassification(dimension, label, classifier.Info().TypeKey)
		} else {
		}
		node.Children.Set(key, child)
		path = append(path, label)
		node = child
		node.Count++
		if !message.Read {
			node.UnreadCount++
		}
	}
	node.Messages = append(node.Messages, message)
	// only append to leaf node
	root.Count++
	if !message.Read {
		root.UnreadCount++
	}
}
