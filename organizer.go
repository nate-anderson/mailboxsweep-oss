package mailsweeper

// A message must have exactly one value for every dimension it is classified by
// example: { "sender": "nate@gmail.com" }, { "sender_domain" : "costco.com" }

type Organizer struct {
	repo        ClassificationRepository
	folders     []Folder
	classifiers []Classifier
}
