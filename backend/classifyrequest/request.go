package classifyrequest

type ClassifyRequest struct {
	Inbox string `dynamodbav:"address"`
	// delete or mark read, only known once the request is executed
	Operation            *string `dynamodbav:"operation"`
	UserUUID             string  `dynamodbav:"user_uuid"`
	RequestedAt          int64   `dynamodbav:"requested_at"`
	CompletedAtUnix      *int64  `dynamodbav:"completed_at_unix"`
	ExecuteStartedAtUnix *int64  `dynamodbav:"execute_started_at_unix"`
	ExecutedAtUnix       *int64  `dynamodbav:"executed_at_unix"`
	// number of messages affected when executed
	Count *int    `dynamodbav:"count"`
	Error *string `dynamodbav:"error"`
}

// ordered statuses allow comparison
// do not serialize this, only get it by calling .Status()
type ClassifyRequestStatus int

const (
	// something went wrong
	StatusErrored ClassifyRequestStatus = iota
	// request is being classified, or is in the queue
	StatusClassifying
	// request has been classified, ready for cleanup
	StatusClassified
	// user has submitted selections, inbox in queue for execute
	StatusExecuting
	// messages have been modified or deleted
	StatusExecuted
)

func (r ClassifyRequest) Status() (status ClassifyRequestStatus, touched int64) {
	if r.Error != nil {
		return StatusErrored, 0
	}

	if r.ExecutedAtUnix != nil {
		return StatusExecuted, *r.ExecutedAtUnix
	}

	if r.ExecuteStartedAtUnix != nil {
		return StatusExecuting, *r.ExecuteStartedAtUnix
	}

	if r.CompletedAtUnix != nil {
		return StatusClassified, *r.CompletedAtUnix
	}

	return StatusClassifying, r.RequestedAt
}

// Equals is used for test assertions
func (r ClassifyRequest) Equals(other ClassifyRequest) bool {
	if r.Inbox != other.Inbox {
		return false
	}

	if r.Operation != other.Operation {
		return false
	}

	if r.UserUUID != other.UserUUID {
		return false
	}

	if r.RequestedAt != other.RequestedAt {
		return false
	}

	if !pointerValsEqual(r.CompletedAtUnix, other.CompletedAtUnix) {
		return false
	}

	if !pointerValsEqual(r.ExecuteStartedAtUnix, other.ExecuteStartedAtUnix) {
		return false
	}

	if !pointerValsEqual(r.ExecutedAtUnix, other.ExecutedAtUnix) {
		return false
	}

	if !pointerValsEqual(r.Count, other.Count) {
		return false
	}

	if !pointerValsEqual(r.Error, other.Error) {
		return false
	}

	return true
}

func pointerValsEqual[T comparable](a, b *T) bool {
	if (a == nil) != (b == nil) {
		return false
	} else if a == nil {
		return true
	}

	return *a == *b
}
