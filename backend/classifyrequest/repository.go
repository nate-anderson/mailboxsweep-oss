package classifyrequest

import (
	"context"
	"fmt"
	"mailboxsweep"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type ClassifyRequestRepository interface {
	Store(ctx context.Context, req ClassifyRequest) error
	List(ctx context.Context, inbox string, onlyIncomplete, consistentRead bool) ([]ClassifyRequest, error)
	MarkCompleted(ctx context.Context, inbox string, startedAt int64, errDetails *string) (*ClassifyRequest, error)
	MarkExecuted(ctx context.Context, inbox string, startedAt int64, operation string, count int, errDetails *string) (*ClassifyRequest, error)
}

type DynamoRequestRepository struct {
	db        *dynamodb.Client
	tableName string
}

func NewDynamoRequestRepository(db *dynamodb.Client, table string) *DynamoRequestRepository {
	return &DynamoRequestRepository{
		db,
		table,
	}
}

func (r DynamoRequestRepository) formatPK(inbox string) string {
	return fmt.Sprintf("CLASSIFY_REQUEST#%s", inbox)
}

func (r DynamoRequestRepository) formatSK(createdAt int64) string {
	return strconv.FormatInt(createdAt, 10)
}

func (r DynamoRequestRepository) buildKey(inbox string, createdAt int64) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		mailboxsweep.DynamoPK: &types.AttributeValueMemberS{
			Value: r.formatPK(inbox),
		},
		mailboxsweep.DynamoSK: &types.AttributeValueMemberS{
			Value: r.formatSK(createdAt),
		},
	}
}

func (r DynamoRequestRepository) Store(ctx context.Context, req ClassifyRequest) error {
	key := r.buildKey(req.Inbox, req.RequestedAt)
	attrs, err := attributevalue.MarshalMap(req)
	if err != nil {
		return err
	}
	item := mailboxsweep.MergeAttributeValueMaps(
		key,
		attrs,
	)

	_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      item,
	})

	return err
}

func (r DynamoRequestRepository) List(ctx context.Context, inbox string, onlyIncomplete, consistentRead bool) ([]ClassifyRequest, error) {
	pk := r.formatPK(inbox)
	var startKey map[string]types.AttributeValue
	var requests []ClassifyRequest
	input := &dynamodb.QueryInput{
		TableName:              &r.tableName,
		KeyConditionExpression: aws.String("#pk = :pk"),
		ExpressionAttributeNames: map[string]string{
			"#pk": mailboxsweep.DynamoPK,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{
				Value: pk,
			},
		},
		ExclusiveStartKey: startKey,
		ConsistentRead:    &consistentRead,
	}
	if onlyIncomplete {
		input.FilterExpression = aws.String("#completed_at_unix = :null OR attribute_not_exists(#completed_at_unix)")
		input.ExpressionAttributeNames["#completed_at_unix"] = "completed_at_unix"
		input.ExpressionAttributeValues[":null"] = &types.AttributeValueMemberNULL{
			Value: true,
		}
	}
	for {
		var chunk []ClassifyRequest
		queryRes, err := r.db.Query(ctx, input)
		if err != nil {
			return requests, err
		}

		if err := attributevalue.UnmarshalListOfMaps(queryRes.Items, &chunk); err != nil {
			return requests, err
		} else {
			requests = append(requests, chunk...)
		}

		if len(queryRes.LastEvaluatedKey) > 0 {
			startKey = queryRes.LastEvaluatedKey
		} else {
			break
		}
	}
	return requests, nil
}

func (r DynamoRequestRepository) MarkCompleted(ctx context.Context, inbox string, startedAt int64, errDetails *string) (*ClassifyRequest, error) {
	key := r.buildKey(inbox, startedAt)
	completed := time.Now().Unix()
	var errDetailsValue types.AttributeValue = &types.AttributeValueMemberNULL{
		Value: true,
	}
	if errDetails != nil {
		errDetailsValue = &types.AttributeValueMemberS{
			Value: *errDetails,
		}
	}
	res, err := r.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		Key:                 key,
		TableName:           &r.tableName,
		UpdateExpression:    aws.String("SET #completed = :completed, #error = :err_details"),
		ConditionExpression: aws.String("#sk = :sk"),
		ExpressionAttributeNames: map[string]string{
			"#completed": "completed_at_unix",
			"#error":     "error",
			"#sk":        mailboxsweep.DynamoSK,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":completed": &types.AttributeValueMemberN{
				Value: strconv.FormatInt(completed, 10),
			},
			":err_details": errDetailsValue,
			":sk": &types.AttributeValueMemberS{
				Value: r.formatSK(startedAt),
			},
		},
		ReturnValues: types.ReturnValueAllNew,
	})
	if err != nil {
		return nil, err
	}

	c := ClassifyRequest{}
	err = attributevalue.UnmarshalMap(res.Attributes, &c)
	return &c, err
}

func (r DynamoRequestRepository) MarkExecuted(ctx context.Context, inbox string, startedAt int64, operation string, count int, errDetails *string) (*ClassifyRequest, error) {
	key := r.buildKey(inbox, startedAt)
	executed := time.Now().Unix()
	var errDetailsValue types.AttributeValue = &types.AttributeValueMemberNULL{
		Value: true,
	}
	if errDetails != nil {
		errDetailsValue = &types.AttributeValueMemberS{
			Value: *errDetails,
		}
	}
	res, err := r.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		Key:                 key,
		TableName:           &r.tableName,
		UpdateExpression:    aws.String("SET #executed = :executed, #error = :err_details, #count = :count, #operation = :operation"),
		ConditionExpression: aws.String("#sk = :sk"),
		ExpressionAttributeNames: map[string]string{
			"#executed":  "executed_at_unix",
			"#error":     "error",
			"#sk":        mailboxsweep.DynamoSK,
			"#count":     "count",
			"#operation": "operation",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":executed": &types.AttributeValueMemberN{
				Value: strconv.FormatInt(executed, 10),
			},
			":err_details": errDetailsValue,
			":sk": &types.AttributeValueMemberS{
				Value: r.formatSK(startedAt),
			},
			":count": &types.AttributeValueMemberN{
				Value: strconv.Itoa(count),
			},
			":operation": &types.AttributeValueMemberS{
				Value: operation,
			},
		},
		ReturnValues: types.ReturnValueAllNew,
	})
	if err != nil {
		return nil, err
	}

	c := ClassifyRequest{}
	err = attributevalue.UnmarshalMap(res.Attributes, &c)
	return &c, err
}
