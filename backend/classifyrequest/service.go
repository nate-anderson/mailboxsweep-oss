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

	// Find the most recent request that has work yet to do
	FindLatestIncompleteRequest(ctx context.Context) (*ClassifyRequest, error)

	// Find the current request to show the user
	FindLatestRequest(ctx context.Context) (*ClassifyRequest, error)

	FindClassifiedRequests(ctx context.Context) ([]ClassifyRequest, error)

	// classification has finished, ready for user to create a change set
	MarkCompletedSuccess(ctx context.Context) error
	MarkCompletedError(ctx context.Context, err error) error

	// mark the request as currently being executed (messages are being modified or deleted)
	MarkExecuting(ctx context.Context, operation string) error

	// change set has been executed, a new scan is needed to do anything else
	MarkExecutedSuccess(ctx context.Context, count int) error
	MarkExecutedError(ctx context.Context, count int, err error) error

	// get sums of messages affected by executed changesets
	SummarizeRequests(ctx context.Context) (*RequestOverview, error)
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

func (s DefaultService) findOpenRequests(ctx context.Context) ([]ClassifyRequest, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	list, err := s.requestRepo.List(ctx, addr, true, false)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, ErrNoClassifyingRequests
	}

	results := []ClassifyRequest{}
	for _, r := range list {
		if r.ExecutedAtUnix != nil {
			continue
		}
		results = append(results, r)
	}

	return results, nil
}

// incurs an expensive consistent read
// this method is used to ensure plan limits are respected
func (s DefaultService) FindClassifiedRequests(ctx context.Context) ([]ClassifyRequest, error) {
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

var (
	ErrNoClassifyingRequests = errors.New("no open classify requests")
	ErrNoIncompleteRequests  = errors.New("no incomplete classify requests")
	ErrNoRequests            = errors.New("no classify requests")
)

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
		return ErrNoClassifyingRequests
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
		return ErrNoClassifyingRequests
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

	// find oldest request (by completion timestamp) that has been completed but not started to execute
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
		} else if oldestReadyRequest.CompletedAtUnix != nil &&
			*req.CompletedAtUnix < *oldestReadyRequest.CompletedAtUnix {
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
		return ErrNoExecutingRequests
	}

	_, err = s.requestRepo.MarkExecuted(ctx, addr, oldestClosedRequest.RequestedAt, count, nil)
	return err
}

var ErrNoExecutingRequests = errors.New("no executing requests")

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
		return ErrNoExecutingRequests
	}

	errMsg := executeError.Error()
	_, err = s.requestRepo.MarkExecuted(ctx, addr, oldestClosedRequest.RequestedAt, count, &errMsg)
	return err
}

func (s DefaultService) FindLatestIncompleteRequest(ctx context.Context) (*ClassifyRequest, error) {
	classifiedRequests, err := s.FindClassifiedRequests(ctx)
	if err != nil {
		return nil, err
	}

	var (
		result      ClassifyRequest
		maxStatus   ClassifyRequestStatus
		lastTouched int64
		found       bool
	)

	for _, request := range classifiedRequests {
		if status, touched := request.Status(); status >= maxStatus && status != StatusExecuted && status != StatusErrored && touched >= lastTouched {
			result = request
			maxStatus = status
			lastTouched = touched
			found = true
		}
	}

	if found {
		return &result, nil
	}

	// if no classified requests, find in-progress
	classifyingRequests, err := s.findOpenRequests(ctx)
	if err != nil {
		return nil, err
	}

	for _, request := range classifyingRequests {
		if status, touched := request.Status(); status > maxStatus &&
			status != StatusExecuted &&
			status != StatusErrored &&
			touched >= lastTouched {
			result = request
			maxStatus = status
			lastTouched = touched
			found = true
		}
	}

	if found {
		return &result, nil
	}

	return nil, ErrNoIncompleteRequests

}

func (s DefaultService) FindLatestRequest(ctx context.Context) (*ClassifyRequest, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	requests, err := s.requestRepo.List(ctx, addr, false, false)
	if err != nil {
		return nil, err
	}

	if len(requests) == 0 {
		return nil, ErrNoRequests
	}

	var (
		latest        ClassifyRequest
		latestTouched int64
	)

	for _, req := range requests {
		if _, touched := req.Status(); touched > latestTouched {
			latest = req
			latestTouched = touched
		}
	}

	return &latest, nil
}

type RequestOverview struct {
	DeletedCount         int
	MarkedReadCount      int
	ExecutedRequestCount int
}

func (s DefaultService) SummarizeRequests(ctx context.Context) (*RequestOverview, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var result RequestOverview

	requests, err := s.requestRepo.List(ctx, addr, false, false)
	if err != nil {
		return nil, err
	}

	for _, r := range requests {
		if r.Count == nil {
			continue
		}
		if r.Operation == nil {
			continue
		}
		switch *r.Operation {
		case "DELETE":
			result.DeletedCount += *r.Count
		case "MARK_READ":
			result.MarkedReadCount += *r.Count
		}
		result.ExecutedRequestCount++
	}

	return &result, nil
}
