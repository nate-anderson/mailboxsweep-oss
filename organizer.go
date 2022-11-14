package mailsweeper

import (
	"fmt"
	"mailsweeper/pkg/syncmap"
)

// A message must have exactly one value for every dimension it is classified by
// example: { "sender": "nate@gmail.com" }, { "sender_domain" : "costco.com" }

type Organizer struct {
	inbox       string
	repo        ClassificationRepository
	folders     []Folder
	classifiers []Classifier
}

func NewOrganizer(inbox string, repo ClassificationRepository, folders []Folder, classifiers []Classifier) *Organizer {
	return &Organizer{
		inbox:       inbox,
		repo:        repo,
		folders:     folders,
		classifiers: classifiers,
	}
}

func (o Organizer) Organize() error {
	/*
		This function should accept a chan Message and process each message as it is read,
		traversing the tree and calling the classifier functions one at a time.
		This reduces the cost of classifying all the messages because it will be done
		concurrently with fetching them.
	*/

	root := &Classification{
		Label:    "All Messages",
		Children: syncmap.New[string, *Classification](),
	}

	m := make(chan Message)
	e := make(chan error)
	for _, f := range o.folders {

		go func() {
			for msg := range m {
				fmt.Println("Read from stream!")
				go o.organizeMessage(msg, root)
			}
		}()

		go f.Messages(1, 100, m, e)
	}

	err := <-e
	if err != nil {
		return err
	}

	o.repo.Store(o.inbox, root)
	return err
}

func (o Organizer) organizeMessage(message Message, parent *Classification) {
	node := parent
	path := []string{}
	for _, classifier := range o.classifiers {
		value := classifier.Classify(message)
		child, ok := node.Children.Get(value)
		if !ok {
			child = NewClassification(value)
			child.Messages = append(child.Messages, message)
		} else {
			child.Messages = append(child.Messages, message)
		}
		node.Children.Set(value, child)
		path = append(path, value)
		node = child
	}
}
