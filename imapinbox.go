package mailsweeper

import (
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type IMAPInbox struct {
	client *client.Client
}

func NewIMAPInbox(client *client.Client) *IMAPInbox {
	return &IMAPInbox{
		client: client,
	}
}

func (i IMAPInbox) Folders() ([]string, error) {
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

func (i *IMAPInbox) Folder(mailbox string) (Folder[uint32], error) {
	mbox, err := i.client.Select(mailbox, false)
	if err != nil {
		return nil, err
	}

	return IMAPFolder{
		client:  i.client,
		mailbox: mbox,
	}, nil
}
