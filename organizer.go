package mailsweeper

import "mailsweeper/pkg/syncmap"

// A message must have exactly one value for every dimension it is classified by
// example: { "sender": "nate@gmail.com" }, { "sender_domain" : "costco.com" }

type Organizer struct {
	repo        ClassificationRepository
	folders     []Folder
	classifiers []Classifier
}

func NewOrganizer(repo ClassificationRepository, folders []Folder, classifiers []Classifier) *Organizer {
	return &Organizer{
		repo:        repo,
		folders:     folders,
		classifiers: classifiers,
	}
}

func (o Organizer) Organize(m chan Message, classifierIndex int) {
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

	for msg := range m {
		o.organizeMessage(msg, root)
	}
}

func (o Organizer) organizeMessage(message Message, parent *Classification) {
	node := parent
	for _, classifier := range o.classifiers {
		value := classifier.Classify(message)
		child, ok := node.Children.Get(value)
		if !ok {
			child = &Classification{
				Label:    value,
				Messages: []Message{message},
			}
		} else {
			child.Messages = append(child.Messages, message)
			node.Children.Set(value, child)
		}
		node.Children.Set(value, child)
		node = child
	}
}
