package mailsweeper

import "time"

type Message struct {
	ID      string
	Subject string
	Sender  Sender
	// message ID being replied to
	InReplyTo string
	SendDate  time.Time
}

type Sender struct {
	Name    string
	Address string
}
