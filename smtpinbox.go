package mailsweeper

import (
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type SMTPInbox struct {
	client *client.Client
}

func NewSMTPInbox(client *client.Client) *SMTPInbox {
	return &SMTPInbox{
		client: client,
	}
}

func (i SMTPInbox) Folders() ([]string, error) {
	mailboxChan := make(chan *imap.MailboxInfo)
	done := make(chan error)
	go func() {
		done <- i.client.List("", "*", mailboxChan)
	}()

	mailboxes := []imap.MailboxInfo{}
	for {
		select {
		case m := <-mailboxChan:
			if m != nil {
				mailboxes = append(mailboxes, *m)
			}
		case err := <-done:
			if err == nil {
				goto BuildRet
			}
			return []string{}, err
		}
	}

BuildRet:
	mailboxNames := make([]string, 0, len(mailboxes))
	for _, m := range mailboxes {
		mailboxNames = append(mailboxNames, m.Name)
	}

	return mailboxNames, nil
}

func (i *SMTPInbox) Folder(mailbox string) (Folder, error) {
	mbox, err := i.client.Select(mailbox, true)
	if err != nil {
		return nil, err
	}

	return SMTPFolder{
		client:  i.client,
		mailbox: mbox,
	}, nil
}
