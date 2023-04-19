package classifyrequest

type ClassifyRequest struct {
	Inbox string `dynamodbav:"address"`
	// delete or mark read, only known once the request is executed
	Operation       *string `dynamodbav:"operation"`
	UserUUID        string  `dynamodbav:"user_uuid"`
	RequestedAt     int64   `dynamodbav:"requested_at"`
	CompletedAtUnix *int64  `dynamodbav:"completed_at_unix"`
	ExecutedAtUnix  *int64  `dynamodbav:"executed_at_unix"`
	// number of messages affected when executed
	Count *int    `dynamodbav:"count"`
	Error *string `dynamodbav:"error"`
}
