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
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
)

type GmailAPIRateLimits struct {
	// rate limiting is configurable because Google doesn't seem to
	// agree with the rate limiter we use
	APICallLimit rate.Limit // 250 by default

	DeleteCallCost       int
	MarkReadCallCost     int
	ListMessagesCallCost int
	GetMessageCallCost   int
}

// Gmail labels are approximated to folders in our system
type GmailFolder struct {
	address string
	label   *gmail.Label
	client  *gmail.Service
	logger  *logrus.Logger

	limits *GmailAPIRateLimits
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
	// rate-limit API calls
	limiter := rate.NewLimiter(f.limits.APICallLimit, f.limits.MarkReadCallCost)
	batches := batch.WithLimit(ids, deleteBatchLimit)

	concurrencyLimit := 5
	workers := make(chan bool, concurrencyLimit)
	errChan := make(chan error)

	wg := sync.WaitGroup{}

	for _, batch := range batches {
		wg.Add(1)
		go func(batch []string) {
			defer wg.Done()
			defer func() {
				<-workers
			}()
			err := limiter.WaitN(ctx, f.limits.MarkReadCallCost)
			workers <- true
			if err != nil {
				errChan <- err
				return
			}

			err = f.client.Users.Messages.BatchDelete(f.address, &gmail.BatchDeleteMessagesRequest{
				Ids: batch,
			}).Context(ctx).Do()
			if err != nil {
				var apiErr *googleapi.Error
				if apiErr.Code == http.StatusForbidden {
					time.Sleep(time.Second * 10)
					return
				}
				errChan <- err
				return
			}
		}(batch)

	}
	wg.Wait()
	return nil
}

const unreadLabel = "UNREAD"

func (f GmailFolder) MarkRead(ctx context.Context, ids []string) error {
	// rate-limit API calls
	limiter := rate.NewLimiter(f.limits.APICallLimit, f.limits.MarkReadCallCost)
	batches := batch.WithLimit(ids, deleteBatchLimit)

	concurrencyLimit := 5
	workers := make(chan bool, concurrencyLimit)
	errChan := make(chan error)

	wg := sync.WaitGroup{}

	for _, batch := range batches {
		wg.Add(1)
		go func(batch []string) {
			defer wg.Done()
			defer func() {
				<-workers
			}()
			err := limiter.WaitN(ctx, f.limits.MarkReadCallCost)
			workers <- true
			if err != nil {
				errChan <- err
				return
			}

			err = f.client.Users.Messages.BatchModify(f.address, &gmail.BatchModifyMessagesRequest{
				RemoveLabelIds: []string{unreadLabel},
				Ids:            batch,
			}).Context(ctx).Do()
			if err != nil {
				var apiErr *googleapi.Error
				if apiErr.Code == http.StatusForbidden {
					time.Sleep(time.Second * 10)
					return
				}
				errChan <- err
				return
			}
		}(batch)
	}

	wg.Wait()
	return nil
}

func (f GmailFolder) Messages(ctx context.Context, start, end uint32, m chan<- Message, e chan<- error) {
	if start >= end {
		e <- fmt.Errorf("folder.Messages called with start >= end")
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
	limiter := rate.NewLimiter(f.limits.APICallLimit, f.limits.ListMessagesCallCost)

	concurrencyLimit := 50
	workers := make(chan bool, concurrencyLimit)

	wg := sync.WaitGroup{}

	for i < end {
		err := limiter.WaitN(ctx, f.limits.ListMessagesCallCost)
		if err != nil {
			e <- fmt.Errorf("rate limit error: %w", err)
			return
		}
		res, err := f.client.Users.Messages.
			List(f.address).
			LabelIds(f.label.Id).
			MaxResults(500).
			PageToken(nextPageToken).
			Context(ctx).
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
				err := limiter.WaitN(ctx, f.limits.GetMessageCallCost)
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
