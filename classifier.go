package mailsweeper

import (
	"fmt"
	"strings"
	"time"
)

type Classifier interface {
	// the classification key for this classifier run
	Key() string
	// classifies messages
	Classify(m Message) string
}

/*
	Classify by sender domain
*/

type SenderDomainClassifier struct{}

func NewSenderDomainClassifier() SenderDomainClassifier {
	return SenderDomainClassifier{}
}

func (c SenderDomainClassifier) Key() string {
	return "Sender Domain"
}

func (c SenderDomainClassifier) Classify(m Message) string {
	return m.Sender.Domain
}

/*
	Classify by sender username
	Assumes messages have already been classified by sender domain
*/

type SenderUsernameClassifier struct{}

func NewSenderUsernameClassifier() SenderUsernameClassifier {
	return SenderUsernameClassifier{}
}

func (c SenderUsernameClassifier) Key() string {
	return "Sender"
}

func (c SenderUsernameClassifier) formatSender(name, address string) string {
	str := ""
	if trimmed := strings.TrimSpace(name); trimmed != "" {
		str += trimmed
	}

	if cleanAddr := strings.TrimSpace(address); cleanAddr != "" {
		str += fmt.Sprintf(" (%s)", cleanAddr)
	}
	return str
}

func (c SenderUsernameClassifier) Classify(m Message) string {
	return c.formatSender(m.Sender.Name, m.Sender.Address)
}

/*
	Classify by a cutoff date
*/

type SendDateClassifier struct {
	cutoff          time.Time
	formattedCutoff string
}

var cutoffDateFormat = "01/02/2006"

func NewSendDateClassifier(cutoff time.Time) SendDateClassifier {
	return SendDateClassifier{
		cutoff:          cutoff,
		formattedCutoff: cutoff.Format(cutoffDateFormat),
	}
}

func (s SendDateClassifier) Key() string {
	return "Cutoff date " + s.formattedCutoff
}

func (s SendDateClassifier) Classify(m Message) string {
	if m.SendDate.Before(s.cutoff) {
		return "Before " + s.formattedCutoff
	}
	return "On or after " + s.formattedCutoff
}
