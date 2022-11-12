package mailsweeper

type Message struct {
	ID      uint32
	Subject string
	Sender  Sender
}

type Sender struct {
	Name    string
	Address string
}
