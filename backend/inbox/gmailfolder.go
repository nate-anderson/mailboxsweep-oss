package inbox

import (
	"fmt"
	"html"
	"mailboxsweep"
	"net/mail"

	"time"

	"github.com/mcnijman/go-emailaddress"
	"google.golang.org/api/gmail/v1"
)

// Gmail labels are approximated to folders in our system
type GmailFolder struct {
	address string
	label   *gmail.Label
	client  *gmail.Service
}

func (f GmailFolder) Name() string {
	return f.label.Name
}

func (f GmailFolder) Count() uint32 {
	return uint32(f.label.MessagesTotal)
}

func (f GmailFolder) UnreadCount() uint32 {
	return uint32(f.label.MessagesUnread)
}

func (f GmailFolder) Delete(ids []string) error {
	err := f.client.Users.Messages.BatchDelete(f.address, &gmail.BatchDeleteMessagesRequest{
		Ids: ids,
	}).Do()
	if err != nil {
		return fmt.Errorf("Gmail BatchDelete failed: %w", err)
	}
	return nil
}

const unreadLabel = "UNREAD"

func (f GmailFolder) MarkRead(ids []string) error {
	err := f.client.Users.Messages.BatchModify(f.address, &gmail.BatchModifyMessagesRequest{
		RemoveLabelIds: []string{unreadLabel},
		Ids:            ids,
	}).Do()
	if err != nil {
		return fmt.Errorf("Gmail BatchModify (MarkRead) failed: %w", err)
	}
	return nil
}

func (f GmailFolder) Messages(start, end uint32, m chan<- Message, e chan<- error) {
	var i uint32 = 0
	var nextPageToken string
	for i < end {
		res, err := f.client.Users.Messages.
			List(f.address).
			LabelIds(f.label.Id).
			PageToken(nextPageToken).
			Do()

		if err != nil {
			e <- err
		}

		nextPageToken = res.NextPageToken

		for _, msg := range res.Messages {
			msgRes, err := f.client.Users.Messages.Get(f.address, msg.Id).Do()
			if err != nil {
				e <- err
				return
			}

			sender, err := parseGmailSender(getHeader("From", msgRes.Payload.Headers))
			if err != nil {
				e <- err
				return
			}

			sendDate := time.Unix(0, msgRes.InternalDate*int64(time.Millisecond))

			m <- Message{
				UID:       msg.Id,
				Folder:    f.label.Name,
				Subject:   getHeader("Subject", msgRes.Payload.Headers),
				Sender:    sender,
				InReplyTo: msg.ThreadId,
				SendDate:  sendDate,
				Read:      !includes(msgRes.LabelIds, "UNREAD"),
			}

			i++
			if i >= end {
				return
			}
		}
	}

}

func getHeader(header string, list []*gmail.MessagePartHeader) string {
	for _, val := range list {
		if val.Name == header {
			return val.Value
		}
	}
	return ""
}

func parseGmailSender(sender string) (Sender, error) {
	addr, err := mail.ParseAddress(html.UnescapeString(sender))
	if err != nil {
		return Sender{}, err
	}

	domain, err := mailboxsweep.DomainFromEmail(addr.Address)
	if err != nil {
		return Sender{}, err
	}

	s := Sender{
		Address: addr.Address,
		Name:    addr.Name,
		Domain:  domain,
	}

	mailAddr, err := emailaddress.Parse(addr.Address)
	if err == nil && mailAddr != nil {
		s.Username = mailAddr.LocalPart
	}

	return s, nil
}

func includes[t comparable](list []t, value t) bool {
	for _, v := range list {
		if value == v {
			return true
		}
	}
	return false
}
