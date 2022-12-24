package login

import "time"

type Link struct {
	UserUUID      string    `dynamodbav:"user_uuid"`
	Token         string    `dynamodbav:"token"`
	Expires       int64     `dynamodbav:"expires_at"`
	Created       time.Time `dynamodbav:"created"`
	InitiatedByIP string    `dynamodbav:"initated_by_ip"`
}
