package mailsweeper

import (
	"fmt"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type IMAPInbox struct {
	address string
	client  *client.Client
}

func NewIMAPInbox(client *client.Client, address string) *IMAPInbox {
	return &IMAPInbox{
		client:  client,
		address: address,
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

func (i IMAPInbox) ValidityKeys() (map[string]uint32, error) {
	folders, err := i.Folders()
	if err != nil {
		return nil, fmt.Errorf("failed getting validity keys: fetching folders: %w", err)
	}

	keys := make(map[string]uint32)

	for _, f := range folders {
		stat, err := i.client.Select(f, true)
		if err != nil {
			return nil, fmt.Errorf("failed getting validity keys: selecting folder %s: %w", f, err)
		}

		keys[f] = stat.UidValidity
	}
	return keys, nil
}

func (i IMAPInbox) Name() string {
	return i.address
}
