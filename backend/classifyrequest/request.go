package classifyrequest

type ClassifyRequest struct {
	Inbox           string  `dynamodbav:"address"`
	UserUUID        string  `dynamodbav:"user_uuid"`
	RequestedAt     int64   `dynamodbav:"requested_at"`
	CompletedAtUnix *int64  `dynamodbav:"completed_at_unix"`
	ExecutedAtUnix  *int64  `dynamodbav:"executed_at_unix"`
	Error           *string `dynamodbav:"error"`
}
