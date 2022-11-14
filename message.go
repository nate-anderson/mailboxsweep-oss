package mailsweeper

import "time"

type Message struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	Sender  Sender `json:"sender"`
	// message ID being replied to
	InReplyTo string    `json:"inReplyTo,omitempty"`
	SendDate  time.Time `json:"sendDate"`
}

type Sender struct {
	Address  string `json:"address"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Domain   string `json:"domain"`
}
