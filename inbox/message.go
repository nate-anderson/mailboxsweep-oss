package inbox

import "time"

type Message struct {
	UID     uint32 `json:"uid"`
	Folder  string `json:"folder"`
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
