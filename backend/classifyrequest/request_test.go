package classifyrequest_test

import (
	"mailboxsweep/assert"
	"mailboxsweep/classifyrequest"
	"testing"
)

type statusTestCase struct {
	request classifyrequest.ClassifyRequest
	status  classifyrequest.ClassifyRequestStatus
}

func ptr[t any](val t) *t {
	return &val
}

func TestClassifyRequest(t *testing.T) {
	pastUnix := int64(1695791162)
	testAddress := "test@mailboxsweep.local"

	testCases := []statusTestCase{
		{
			request: classifyrequest.ClassifyRequest{
				Inbox:       testAddress,
				RequestedAt: pastUnix,
			},
			status: classifyrequest.StatusClassifying,
		},
		{
			request: classifyrequest.ClassifyRequest{
				Inbox:           testAddress,
				RequestedAt:     pastUnix - 10,
				CompletedAtUnix: &pastUnix,
			},
			status: classifyrequest.StatusClassified,
		},
		{
			request: classifyrequest.ClassifyRequest{
				Inbox:                testAddress,
				RequestedAt:          pastUnix - 20,
				CompletedAtUnix:      ptr(pastUnix - 10),
				ExecuteStartedAtUnix: &pastUnix,
			},
			status: classifyrequest.StatusExecuting,
		},
		{
			request: classifyrequest.ClassifyRequest{
				Inbox:                testAddress,
				RequestedAt:          pastUnix - 30,
				CompletedAtUnix:      ptr(pastUnix - 20),
				ExecuteStartedAtUnix: ptr(pastUnix - 10),
				ExecutedAtUnix:       &pastUnix,
			},
			status: classifyrequest.StatusExecuted,
		},

		{
			request: classifyrequest.ClassifyRequest{
				Inbox:       testAddress,
				RequestedAt: pastUnix,
				Error:       ptr("error"),
			},
			status: classifyrequest.StatusErrored,
		},
		{
			request: classifyrequest.ClassifyRequest{
				Inbox:           testAddress,
				RequestedAt:     pastUnix - 10,
				CompletedAtUnix: &pastUnix,
				Error:           ptr("error"),
			},
			status: classifyrequest.StatusErrored,
		},
		{
			request: classifyrequest.ClassifyRequest{
				Inbox:                testAddress,
				RequestedAt:          pastUnix - 20,
				CompletedAtUnix:      ptr(pastUnix - 10),
				ExecuteStartedAtUnix: &pastUnix,
				Error:                ptr("error"),
			},
			status: classifyrequest.StatusErrored,
		},
		{
			request: classifyrequest.ClassifyRequest{
				Inbox:                testAddress,
				RequestedAt:          pastUnix - 30,
				CompletedAtUnix:      ptr(pastUnix - 20),
				ExecuteStartedAtUnix: ptr(pastUnix - 10),
				ExecutedAtUnix:       &pastUnix,
				Error:                ptr("error"),
			},
			status: classifyrequest.StatusErrored,
		},
	}

	for _, testCase := range testCases {
		act, touched := testCase.request.Status()
		assert.That(t, act).Equals(testCase.status)
		if testCase.request.Error != nil {
			assert.That(t, touched).Equals(0)
		} else {
			assert.That(t, touched).Equals(pastUnix)
		}
	}
}
