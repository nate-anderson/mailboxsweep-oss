package inbox

import (
	"context"
	"fmt"
	"mailboxsweep/pkg/slice"
	"net/mail"
	"regexp"
	"strconv"
	"strings"

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

func (f IMAPFolder) UnreadCount() uint32 {
	return f.mailbox.Unseen
}

func (f IMAPFolder) Messages(ctx context.Context, start, end uint32, m chan<- Message, e chan<- error) {
	seq := imap.SeqSet{}
	seq.AddRange(start, end)

	messageChan := make(chan *imap.Message)
	done := make(chan error)
	go func() {
		// set the components fetched here
		done <- f.client.Fetch(&seq, []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchUid,
			imap.FetchFlags,
		}, messageChan)
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

var (
	openingAngleBracket = regexp.MustCompile(`^\s*\<`)
	closingAngleBracket = regexp.MustCompile(`\>\s*$`)
)

func sanitize(address string) string {
	clean := strings.TrimSpace(address)
	if !strings.HasPrefix(clean, "<") {
		return clean
	}
	clean = openingAngleBracket.ReplaceAllLiteralString(address, "")
	clean = closingAngleBracket.ReplaceAllLiteralString(clean, "")
	return clean
}

// @TODO: urgent
// this function is not parsing addresses with < > correctly
// domain classifier is failing
func imapToMessage(i imap.Message, folder string) (Message, error) {
	var sender *imap.Address
	if len(i.Envelope.Sender) >= 1 {
		sender = i.Envelope.Sender[0]
	} else if len(i.Envelope.From) >= 1 {
		sender = i.Envelope.From[0]
	} else {
		return Message{}, fmt.Errorf("message %s doesn't have a SENDER or FROM address", i.Envelope.MessageId)
	}
	senderAddress := sanitize(sender.Address())

	var imapSenderName string
	parsed, err := mail.ParseAddress(senderAddress)
	if err == nil && parsed != nil {
		imapSenderName = parsed.Name
	}
	var username, domain string
	mailAddr, err := emailaddress.Parse(senderAddress)
	if err == nil && mailAddr != nil {
		username = mailAddr.LocalPart
		domain = mailAddr.Domain
	}
	return Message{
		Subject:   i.Envelope.Subject,
		UID:       strconv.FormatUint(uint64(i.Uid), 10),
		Folder:    folder,
		InReplyTo: i.Envelope.InReplyTo,
		Sender: Sender{
			Name: firstNotEqual(
				"",
				sender.PersonalName,
				sender.MailboxName,
				imapSenderName,
				username,
				senderAddress,
			),
			Address:  senderAddress,
			Domain:   domain,
			Username: username,
		},
		SendDate: i.Envelope.Date,
		Read:     slice.Contains(i.Flags, imap.SeenFlag),
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

func (f IMAPFolder) MarkRead(uids []string) error {
	readOnly := f.mailbox.ReadOnly
	set := imap.SeqSet{}
	for _, uid := range uids {
		u, err := strconv.ParseUint(uid, 10, 32)
		if err != nil {
			return err
		}
		set.AddNum(uint32(u))
	}

	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []any{imap.SeenFlag}
	err := f.client.UidStore(&set, item, flags, nil)
	if err != nil {
		return fmt.Errorf("delete (UidStore) failed: %w", err)
	}

	// Close propagates changes and deselects the client mailbox
	err = f.client.Close()
	if err != nil {
		return fmt.Errorf("failed to close and push changes to IMAP mailbox: %w", err)
	}

	// reselect for continued use
	f.mailbox, err = f.client.Select(f.name, readOnly)
	return err
}
