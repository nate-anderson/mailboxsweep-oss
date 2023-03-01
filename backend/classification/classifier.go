package classification

import (
	"fmt"
	"mailboxsweep/inbox"
	"regexp"
	"strings"
	"time"
)

type Classifier interface {
	// the object key/path entry to use
	Key(m inbox.Message) string
	// the string to display to users
	Classify(m inbox.Message) string
	// what is this thing classifying by?
	Dimension() string
	Serializeable() SerializableClassifier
	// Display info about the classifier - should only return static data
	Info() ClassifierInfo
}

type InputFormat string

const (
	InputDate   InputFormat = "DATE"
	InputString InputFormat = "STRING"
)

// used for listing available classifiers to API consumers
type ClassifierInfo struct {
	Name        string
	TypeKey     string
	Description string
	TakesInput  bool
	InputLabel  *string
	InputFormat *InputFormat
}

/*
	Classify by sender domain
*/

type SenderDomainClassifier struct{}

func NewSenderDomainClassifier() SenderDomainClassifier {
	return SenderDomainClassifier{}
}

func (c SenderDomainClassifier) Key(m inbox.Message) string {
	return fmt.Sprintf("domain=%s", m.Sender.Domain)
}

func (c SenderDomainClassifier) Classify(m inbox.Message) string {
	return m.Sender.Domain
}

func (c SenderDomainClassifier) Dimension() string {
	return "sender_domain"
}

const TypeKeySenderDomain = "SENDER_DOMAIN"

func (c SenderDomainClassifier) Serializeable() SerializableClassifier {
	return SerializableClassifier{
		TypeKey:   TypeKeySenderDomain,
		InitValue: "",
	}
}

func (c SenderDomainClassifier) Info() ClassifierInfo {
	return ClassifierInfo{
		Name:        "Sender domain",
		TypeKey:     TypeKeySenderDomain,
		Description: "Classify messages by the website of the sender",
		TakesInput:  false,
	}
}

/*
	Classify by sender username
	Assumes messages have already been classified by sender domain
*/

type SenderUsernameClassifier struct{}

func NewSenderUsernameClassifier() SenderUsernameClassifier {
	return SenderUsernameClassifier{}
}

func (c SenderUsernameClassifier) Key(m inbox.Message) string {
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

func (c SenderUsernameClassifier) Classify(m inbox.Message) string {
	return c.formatSender(m.Sender.Name, m.Sender.Address)
}

func (c SenderUsernameClassifier) Dimension() string {
	return "sender"
}

const TypeKeySenderUsername = "SENDER_USERNAME"

func (c SenderUsernameClassifier) Serializeable() SerializableClassifier {
	return SerializableClassifier{
		TypeKey:   TypeKeySenderUsername,
		InitValue: "",
	}
}

func (c SenderUsernameClassifier) Info() ClassifierInfo {
	return ClassifierInfo{
		Name:        "Sender email",
		TypeKey:     TypeKeySenderUsername,
		Description: "Classify messages by the sender",
		TakesInput:  false,
	}
}

/*
	Classify by a cutoff date
*/

type SendDateClassifier struct {
	cutoff          time.Time
	formattedCutoff string
}

var cutoffDateFormat = "01-02-2006"

func NewSendDateClassifier(cutoffStr string) (SendDateClassifier, error) {
	cutoff, err := time.Parse(cutoffStr, cutoffDateFormat)
	if err != nil {
		return SendDateClassifier{}, fmt.Errorf("send date classifer input must be of format %s: %w", cutoffDateFormat, err)
	}
	return SendDateClassifier{
		cutoff:          cutoff,
		formattedCutoff: cutoff.Format(cutoffDateFormat),
	}, nil
}

func (s SendDateClassifier) Key(m inbox.Message) string {
	if m.SendDate.Before(s.cutoff) {
		return "before_" + s.formattedCutoff
	}
	return "on_or_after_" + s.formattedCutoff
}

func (s SendDateClassifier) Classify(m inbox.Message) string {
	if m.SendDate.Before(s.cutoff) {
		return "Before " + s.formattedCutoff
	}
	return "On or after " + s.formattedCutoff
}

func (s SendDateClassifier) Dimension() string {
	return fmt.Sprintf("send_date_before_or_after_%s", s.formattedCutoff)
}

const TypeKeySendDate = "SEND_DATE"

func (s SendDateClassifier) Serializeable() SerializableClassifier {
	return SerializableClassifier{
		TypeKey:   TypeKeySendDate,
		InitValue: s.formattedCutoff,
	}
}

func (s SendDateClassifier) Info() ClassifierInfo {
	return ClassifierInfo{
		Name:        "Send date",
		TypeKey:     TypeKeySendDate,
		Description: "Divide messages before and after a given date",
		TakesInput:  true,
		InputLabel:  ptr("Filter date"),
		InputFormat: ptr(InputDate),
	}
}

const TypeKeySubjectKeyword = "SUBJECT_KEYWORD"

type SubjectKeywordClassifier struct {
	keyword string
	regex   *regexp.Regexp
}

func NewSubjectKeywordClassifier(keyword string) (SubjectKeywordClassifier, error) {
	clean := strings.TrimSpace(strings.ToLower(keyword))
	if len(clean) < 3 {
		return SubjectKeywordClassifier{}, fmt.Errorf("invalid keyword '%s': must be at least 3 characters", keyword)
	}

	expr := fmt.Sprintf("(^|\\s+|['?.,!@&*=+-_~:;\"></^()%%])%s($|\\s+|['?.,!@&*=+-_~:;\"></^()%%])", clean)
	regex, err := regexp.Compile(expr)
	if err != nil {
		return SubjectKeywordClassifier{}, err
	}
	return SubjectKeywordClassifier{
		keyword: clean,
		regex:   regex,
	}, nil
}

func (c SubjectKeywordClassifier) Key(m inbox.Message) string {
	if c.regex.MatchString(strings.ToLower(m.Subject)) {
		return fmt.Sprintf("contains_keyword_%s", c.keyword)
	}
	return fmt.Sprintf("not_contains_keyword_%s", c.keyword)
}

func (c SubjectKeywordClassifier) Classify(m inbox.Message) string {
	if c.regex.MatchString(strings.ToLower(m.Subject)) {
		return fmt.Sprintf("Subject includes \"%s\"", c.keyword)
	}
	return fmt.Sprintf("Subject does not include \"%s\"", c.keyword)
}

func (c SubjectKeywordClassifier) Dimension() string {
	return "subject_keyword_" + c.keyword
}

func (c SubjectKeywordClassifier) Serializeable() SerializableClassifier {
	return SerializableClassifier{
		TypeKey:   TypeKeySubjectKeyword,
		InitValue: c.keyword,
	}
}

func (c SubjectKeywordClassifier) Info() ClassifierInfo {
	return ClassifierInfo{
		Name:        "Subject keyword",
		TypeKey:     TypeKeySubjectKeyword,
		Description: "Classify messages by subject keyword",
		TakesInput:  true,
		InputFormat: ptr(InputString),
		InputLabel:  ptr("Keyword"),
	}
}

// Classifier serialization

type SerializableClassifier struct {
	TypeKey   string `json:"type_key"`
	InitValue string `json:"init_value"`
}

func (s SerializableClassifier) Classifier() (Classifier, error) {
	switch s.TypeKey {
	case TypeKeySenderDomain:
		return NewSenderDomainClassifier(), nil
	case TypeKeySenderUsername:
		return NewSenderUsernameClassifier(), nil
	case TypeKeySendDate:
		return NewSendDateClassifier(s.InitValue)
	case TypeKeySubjectKeyword:
		return NewSubjectKeywordClassifier(s.InitValue)
	default:
		return nil, fmt.Errorf("unknown type key for serialized classifier '%s'", s.TypeKey)
	}
}

var allClassifiers = []Classifier{
	SenderDomainClassifier{},
	SenderUsernameClassifier{},
	SendDateClassifier{},
	SubjectKeywordClassifier{},
}

func ptr[t any](v t) *t {
	return &v
}
