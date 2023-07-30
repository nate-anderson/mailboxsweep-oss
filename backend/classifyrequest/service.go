package classifyrequest

import (
	"context"
	"errors"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/inboxuser"
	"time"
)

type Service interface {
	RecordRequest(ctx context.Context) error
	FindOpenRequests(ctx context.Context) ([]ClassifyRequest, error)
	FindCompletedRequests(ctx context.Context) ([]ClassifyRequest, error)

	// classification has finished, ready for user to create a change set
	MarkCompletedSuccess(ctx context.Context) error
	MarkCompletedError(ctx context.Context, err error) error

	// mark the request as currently being executed (messages are being modified or deleted)
	MarkExecuting(ctx context.Context, operation string) error

	// change set has been executed, a new scan is needed to do anything else
	MarkExecutedSuccess(ctx context.Context, count int) error
	MarkExecutedError(ctx context.Context, count int, err error) error
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
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	user, err := s.userService.FindByAddress(ctx, addr)
	if err != nil {
		return fmt.Errorf("failed retrieving user: %w", err)
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

	timeSinceLastComplete := time.Since(lastCompleteRequest)
	if timeSinceLastComplete < s.timeBetweenClosedRequests {
		return fmt.Errorf("cannot submit another request until %s after last completed request (%s)", s.timeBetweenClosedRequests.String(), lastCompleteRequest.Format(time.Stamp))
	}

	timeSinceLastIncomplete := time.Since(lastStartedRequest)
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
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	list, err := s.requestRepo.List(ctx, addr, true, false)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, ErrNoOpenRequests
	}

	return list, nil
}

// incurs an expensive consistent read
// this method is used to ensure plan limits are respected
func (s DefaultService) FindCompletedRequests(ctx context.Context) ([]ClassifyRequest, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	list, err := s.requestRepo.List(ctx, addr, false, true)
	if err != nil {
		return nil, err
	}

	completed := []ClassifyRequest{}
	for _, req := range list {
		if req.CompletedAtUnix == nil {
			continue
		}
		completed = append(completed, req)
	}

	return completed, nil
}

var ErrNoOpenRequests = errors.New("no open classify requests")

func (s DefaultService) MarkCompletedSuccess(ctx context.Context) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
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
			break
		}
	}
	if selected.Inbox == "" {
		return ErrNoOpenRequests
	}

	_, err = s.requestRepo.MarkCompleted(ctx, addr, selected.RequestedAt, nil)
	return err
}

func (s DefaultService) MarkCompletedError(ctx context.Context, classifyErr error) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
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
			break
		}
	}
	if selected.Inbox == "" {
		return ErrNoOpenRequests
	}

	msg := classifyErr.Error()
	_, err = s.requestRepo.MarkCompleted(ctx, addr, selected.RequestedAt, &msg)
	return err
}

// Mark the most recently completed classify request as executing
func (s DefaultService) MarkExecuting(ctx context.Context, operation string) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	completedRequests, err := s.requestRepo.ListCompleted(ctx, addr, true)
	if err != nil {
		return err
	}

	// find oldest request that has been completed but not started to execute
	var oldestReadyRequest ClassifyRequest
	found := false

	for _, req := range completedRequests {
		if req.CompletedAtUnix == nil {
			continue
		}

		if req.ExecuteStartedAtUnix != nil || req.ExecutedAtUnix != nil {
			continue
		}

		if !found {
			oldestReadyRequest = req
			found = true
		} else if *req.ExecuteStartedAtUnix < *oldestReadyRequest.ExecuteStartedAtUnix {
			oldestReadyRequest = req
		}
	}

	if !found {
		return errors.New("no requests ready to start executing were found")
	}

	_, err = s.requestRepo.MarkExecuting(ctx, addr, oldestReadyRequest.RequestedAt, operation)

	return err
}

func (s DefaultService) MarkExecutedSuccess(ctx context.Context, count int) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	requests, err := s.requestRepo.ListCompleted(ctx, addr, true)
	if err != nil {
		return err
	}

	// find oldest request that has been completed but not executed
	var oldestClosedRequest ClassifyRequest
	found := false
	for _, req := range requests {
		if req.CompletedAtUnix == nil {
			continue
		}

		if req.ExecuteStartedAtUnix == nil {
			continue
		}

		if req.ExecutedAtUnix != nil {
			continue
		}

		if !found {
			oldestClosedRequest = req
			found = true
		} else if *req.CompletedAtUnix < *oldestClosedRequest.CompletedAtUnix {
			oldestClosedRequest = req
		}
	}

	if !found {
		return errors.New("no completed requests found")
	}

	_, err = s.requestRepo.MarkExecuted(ctx, addr, oldestClosedRequest.RequestedAt, count, nil)
	return err
}

func (s DefaultService) MarkExecutedError(ctx context.Context, count int, executeError error) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	if executeError == nil {
		return errors.New("nil error passed to MarkExecutedError")
	}

	requests, err := s.requestRepo.List(ctx, addr, false, true)
	if err != nil {
		return err
	}

	// find oldest request that has been completed but not executed
	var oldestClosedRequest ClassifyRequest
	found := false
	for _, req := range requests {
		if req.CompletedAtUnix == nil || req.ExecuteStartedAtUnix == nil {
			continue
		}

		if req.ExecutedAtUnix != nil {
			continue
		}

		if !found {
			oldestClosedRequest = req
			found = true
		} else if *req.CompletedAtUnix < *oldestClosedRequest.CompletedAtUnix {
			oldestClosedRequest = req
		}
	}

	if !found {
		return errors.New("no completed requests found")
	}

	errMsg := executeError.Error()
	_, err = s.requestRepo.MarkExecuted(ctx, addr, oldestClosedRequest.RequestedAt, count, &errMsg)
	return err
}
