package inbox

import (
	"fmt"

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
	res, err := f.client.Users.Messages.List(f.address).LabelIds(f.label.Id).Do()
	if err != nil {
		e <- err
	}

	for _, msg := range res.Messages {
		m <- Message{
			UID: msg.Id,
		}
	}
}
