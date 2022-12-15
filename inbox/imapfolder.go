package inbox

import (
	"fmt"
	"net/mail"
	"strconv"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/mcnijman/go-emailaddress"
)

type IMAPFolder struct {
	name    string
	client  *client.Client
	mailbox *imap.MailboxStatus
}

func (f IMAPFolder) Count() uint32 {
	return f.mailbox.Messages
}

func (f IMAPFolder) Messages(start, end uint32, m chan<- Message, e chan<- error) {
	seq := imap.SeqSet{}
	seq.AddRange(start, end)

	messageChan := make(chan *imap.Message)
	done := make(chan error)
	go func() {
		// set the components fetched here
		done <- f.client.Fetch(&seq, []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid, imap.FetchFlags}, messageChan)
	}()

	for {
		select {
		case msg := <-messageChan:
			if msg != nil {
				message, err := imapToMessage(*msg, f.mailbox.Name)
				if err != nil {
					e <- err
				}
				m <- message
			}
		case err := <-done:
			e <- err
		}
	}
}

func imapToMessage(i imap.Message, folder string) (Message, error) {
	var sender *imap.Address
	if len(i.Envelope.Sender) >= 1 {
		sender = i.Envelope.Sender[0]
	} else if len(i.Envelope.From) >= 1 {
		sender = i.Envelope.From[0]
	} else {
		return Message{}, fmt.Errorf("message %s doesn't have a SENDER or FROM address", i.Envelope.MessageId)
	}
	var imapSenderName string
	senderEmail := sender.Address()
	parsed, err := mail.ParseAddress(senderEmail)
	if err == nil && parsed != nil {
		imapSenderName = parsed.Name
	}
	var username, domain string
	mailAddr, err := emailaddress.Parse(senderEmail)
	if err == nil && mailAddr != nil {
		username = mailAddr.LocalPart
		domain = mailAddr.Domain
	}
	return Message{
		Subject:   i.Envelope.Subject,
		UID:       i.Uid,
		Folder:    folder,
		InReplyTo: i.Envelope.InReplyTo,
		Sender: Sender{
			Name: firstNotEqual(
				"",
				sender.PersonalName,
				sender.MailboxName,
				imapSenderName,
				username,
				sender.Address(),
			),
			Address:  sender.Address(),
			Domain:   domain,
			Username: username,
		},
		SendDate: i.Envelope.Date,
	}, nil
}

// returns the first value in list not equal to not
// if all values equal not, it returns not
func firstNotEqual[T comparable](not T, list ...T) T {
	for _, item := range list {
		if item != not {
			return item
		}
	}
	return not
}

// @TODO: test this
func (f IMAPFolder) Delete(uids []string) error {
	set := imap.SeqSet{}
	for _, uid := range uids {
		u, err := strconv.ParseUint(uid, 10, 32)
		if err != nil {
			return err
		}
		set.AddNum(uint32(u))
	}
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []any{imap.DeletedFlag}
	err := f.client.UidStore(&set, item, flags, nil)
	if err != nil {
		return fmt.Errorf("delete (UidStore) failed: %w", err)
	}

	if err = f.client.Expunge(nil); err != nil {
		return fmt.Errorf("failed to expunge: %w", err)
	}
	return nil
}

func (f IMAPFolder) Name() string {
	return f.name
}
