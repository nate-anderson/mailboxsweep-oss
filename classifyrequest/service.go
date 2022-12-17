package classifyrequest

import (
	"context"
	"errors"
	"fmt"
	"mailboxsweep/inboxuser"
	"mailboxsweep/session"
	"time"
)

type Service interface {
	RecordRequest(ctx context.Context) error
	FindOpenRequests(ctx context.Context) ([]ClassifyRequest, error)
	CompleteSuccess(ctx context.Context) error
	CompleteError(ctx context.Context, err error) error
}

type DefaultService struct {
	userService               inboxuser.Service
	requestRepo               ClassifyRequestRepository
	timeBetweenOpenRequests   time.Duration
	timeBetweenClosedRequests time.Duration
}

func NewDefaultService(
	userService inboxuser.Service,
	requestRepo ClassifyRequestRepository,
	timeBetweenOpenRequests time.Duration,
	timeBetweenClosedRequests time.Duration,
) *DefaultService {
	return &DefaultService{
		userService:               userService,
		requestRepo:               requestRepo,
		timeBetweenOpenRequests:   timeBetweenOpenRequests,
		timeBetweenClosedRequests: timeBetweenClosedRequests,
	}
}

func (s DefaultService) RecordRequest(ctx context.Context) error {
	addr, err := session.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	user, err := s.userService.FindByAddress(ctx, addr)
	if err != nil {
		return err
	}

	requests, err := s.requestRepo.List(ctx, addr, false, true)
	if err != nil {
		return err
	}

	var lastStartedRequest, lastCompleteRequest time.Time
	for _, req := range requests {
		if req.CompletedAtUnix != nil {
			completed := time.Unix(*req.CompletedAtUnix, 0)
			if completed.After(lastCompleteRequest) {
				lastCompleteRequest = completed
			}
		} else {
			started := time.Unix(req.RequestedAt, 0)
			if started.After(lastStartedRequest) {
				lastStartedRequest = started
			}
		}
	}

	timeSinceLastComplete := time.Now().Sub(lastCompleteRequest)
	if timeSinceLastComplete < s.timeBetweenClosedRequests {
		return fmt.Errorf("cannot submit another request until %s after last completed request (%s)", s.timeBetweenClosedRequests.String(), lastCompleteRequest.Format(time.Stamp))
	}

	timeSinceLastIncomplete := time.Now().Sub(lastStartedRequest)
	if timeSinceLastIncomplete < s.timeBetweenOpenRequests {
		return fmt.Errorf("cannot submit another request: a request has been open for less than %s", s.timeBetweenOpenRequests.String())
	}

	req := ClassifyRequest{
		Inbox:       addr,
		UserUUID:    user.UUID,
		RequestedAt: time.Now().Unix(),
	}
	return s.requestRepo.Store(ctx, req)
}

func (s DefaultService) FindOpenRequests(ctx context.Context) ([]ClassifyRequest, error) {
	addr, err := session.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return s.requestRepo.List(ctx, addr, true, false)
}

var ErrNoOpenRequests = errors.New("no open classify requests")

func (s DefaultService) CompleteSuccess(ctx context.Context) error {
	addr, err := session.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	openRequests, err := s.requestRepo.List(ctx, addr, true, true)
	if err != nil {
		return err
	}

	var selected ClassifyRequest
	for _, req := range openRequests {
		if req.CompletedAtUnix == nil {
			selected = req
		}
	}
	if selected.Inbox == "" {
		return ErrNoOpenRequests
	}

	_, err = s.requestRepo.Update(ctx, addr, time.Now().Unix(), nil)
	return err
}

func (s DefaultService) CompleteError(ctx context.Context, classifyErr error) error {
	addr, err := session.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	openRequests, err := s.requestRepo.List(ctx, addr, true, true)
	if err != nil {
		return err
	}

	var selected ClassifyRequest
	for _, req := range openRequests {
		if req.CompletedAtUnix == nil {
			selected = req
		}
	}
	if selected.Inbox == "" {
		return ErrNoOpenRequests
	}

	msg := classifyErr.Error()
	_, err = s.requestRepo.Update(ctx, addr, time.Now().Unix(), &msg)
	return err
}
