package mailsweeper

import (
	"fmt"
	"strings"
	"time"
)

type Classifier interface {
	// the object key/path entry to use
	Key(m Message) string
	// the string to display to users
	Classify(m Message) string
	// what is this thing classifying by?
	Dimension() string
}

/*
	Classify by sender domain
*/

type SenderDomainClassifier struct{}

func NewSenderDomainClassifier() SenderDomainClassifier {
	return SenderDomainClassifier{}
}

func (c SenderDomainClassifier) Key(m Message) string {
	return fmt.Sprintf("domain=%s", m.Sender.Domain)
}

func (c SenderDomainClassifier) Classify(m Message) string {
	return m.Sender.Domain
}

func (c SenderDomainClassifier) Dimension() string {
	return "sender_domain"
}

/*
	Classify by sender username
	Assumes messages have already been classified by sender domain
*/

type SenderUsernameClassifier struct{}

func NewSenderUsernameClassifier() SenderUsernameClassifier {
	return SenderUsernameClassifier{}
}

func (c SenderUsernameClassifier) Key(m Message) string {
	return "sender=" + m.Sender.Address
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

func (c SenderUsernameClassifier) Dimension() string {
	return "sender"
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

func (s SendDateClassifier) Key(m Message) string {
	if m.SendDate.Before(s.cutoff) {
		return "before_" + s.formattedCutoff
	}
	return "on_or_after_" + s.formattedCutoff
}

func (s SendDateClassifier) Classify(m Message) string {
	if m.SendDate.Before(s.cutoff) {
		return "Before " + s.formattedCutoff
	}
	return "On or after " + s.formattedCutoff
}

func (s SendDateClassifier) Dimension() string {
	return fmt.Sprintf("send_date_before_or_after_%s", s.formattedCutoff)
}
