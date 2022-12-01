package inbox

import (
	"fmt"
	"mailboxsweep/pkg/slice"

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
			// don't include noselect folders here,
			// there's no use for them in this application
			if m != nil && !slice.Contains(m.Attributes, attributeNoSelect) {
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
		name:    mailbox,
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

	selected := i.client.Mailbox()
	var (
		prevName     = selected.Name
		prevReadOnly = selected.ReadOnly
	)

	for _, f := range folders {
		stat, err := i.client.Select(f, true)
		if err != nil {
			return nil, fmt.Errorf("failed getting validity keys: selecting folder %s: %w", f, err)
		}

		keys[f] = stat.UidValidity
	}

	i.client.Select(prevName, prevReadOnly)
	return keys, nil
}

func (i IMAPInbox) Name() string {
	return i.address
}

// IMAP constants
const (
	attributeNoSelect = `\Noselect`
)
