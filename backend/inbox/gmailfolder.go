package inbox

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log"
	"mailboxsweep"
	"mailboxsweep/pkg/batch"
	"net/http"
	"net/mail"
	"strings"
	"sync"

	"time"

	"github.com/mcnijman/go-emailaddress"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
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

const deleteBatchLimit = 1000

func (f GmailFolder) Delete(ctx context.Context, ids []string) error {
	batches := batch.WithLimit(ids, deleteBatchLimit)
	group, groupCtx := errgroup.WithContext(ctx)
	for _, batch := range batches {
		b := batch
		group.Go(func() error {
			err := f.client.Users.Messages.BatchDelete(f.address, &gmail.BatchDeleteMessagesRequest{
				Ids: b,
			}).Context(groupCtx).Do()
			if err != nil {
				return fmt.Errorf("Gmail BatchDelete failed: %w", err)
			}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return err
	}
	return nil
}

const unreadLabel = "UNREAD"

func (f GmailFolder) MarkRead(ctx context.Context, ids []string) error {
	batches := batch.WithLimit(ids, deleteBatchLimit)
	group, groupCtx := errgroup.WithContext(ctx)
	for _, batch := range batches {
		b := batch
		err := f.client.Users.Messages.BatchModify(f.address, &gmail.BatchModifyMessagesRequest{
			RemoveLabelIds: []string{unreadLabel},
			Ids:            b,
		}).Context(groupCtx).Do()
		if err != nil {
			return fmt.Errorf("Gmail BatchModify (MarkRead) failed: %w", err)
		}
	}
	if err := group.Wait(); err != nil {
		return err
	}
	return nil
}

func (f GmailFolder) Messages(ctx context.Context, start, end uint32, m chan<- Message, e chan<- error) {
	if start >= end {
		e <- fmt.Errorf("folder.Messages called with start == end")
		return
	}

	if end == 0 {
		e <- fmt.Errorf("folder.Messages called with zero end")
	}

	var i uint32 = 0
	var nextPageToken string

	// Gmail API allows up to 50 calls per second of the APIs used here
	// (250 units/second, 5 units/call)
	// https://developers.google.com/gmail/api/reference/quota
	callCost := 5
	limiter := rate.NewLimiter(250, callCost)

	concurrencyLimit := 50
	workers := make(chan bool, concurrencyLimit)

	wg := sync.WaitGroup{}

	for i < end {
		err := limiter.WaitN(ctx, callCost)
		if err != nil {
			e <- fmt.Errorf("rate limit error: %w", err)
			return
		}
		res, err := f.client.Users.Messages.
			List(f.address).
			LabelIds(f.label.Id).
			MaxResults(500).
			PageToken(nextPageToken).
			Do()

		if err != nil {
			var apiErr *googleapi.Error
			if errors.As(err, &apiErr) {
				// cool off
				if apiErr.Code == http.StatusForbidden {
					time.Sleep(time.Second * 10)
					continue
				}
			}
			e <- err
		}

		nextPageToken = res.NextPageToken

		wg.Add(len(res.Messages))
		for _, msg := range res.Messages {
			workers <- true
			i++
			go func(msg gmail.Message) {
				defer func() {
					wg.Done()
					<-workers
				}()
				err := limiter.WaitN(ctx, callCost)
				if err != nil {
					e <- err
					return
				}
				msgRes, err := f.client.Users.
					Messages.
					Get(f.address, msg.Id).
					Format("metadata").
					Do()
				if err != nil {
					e <- err
					return
				}

				sender, err := getGmailSender(msgRes.Payload.Headers)
				if err != nil {
					log.Printf("error parsing gmail sender %s for message %s (account %s): %s", getHeader("From", msgRes.Payload.Headers), msg.Id, f.address, err)
					// if there's no sender, log errors and skip. We just won't let the user act on this message from our app.
					return
				}

				// InternalDate is the date Gmail received the message, not the date it was sent
				// but it's close enough. the header dates have inconsistent formatting and are sometimes missing.
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
			}(*msg)
		}
	}

	wg.Wait()
	// send completion signal
	e <- nil
}

func getHeader(header string, list []*gmail.MessagePartHeader) string {
	lowerHeader := strings.ToLower(header)
	for _, val := range list {
		if strings.ToLower(val.Name) == lowerHeader {
			return val.Value
		}
	}
	return ""
}

func getGmailSender(headers []*gmail.MessagePartHeader) (Sender, error) {
	if sender, err := parseGmailSender(getHeader("from", headers)); err == nil {
		return sender, nil
	} else {
		return Sender{}, fmt.Errorf("Failed to parse sender header: %w", err)
	}
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
