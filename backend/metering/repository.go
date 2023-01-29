package metering

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

type UsageRepository interface {
	// update the current day's usage
	// update often, reading bytes used value from memory whenever possible
	Update(ctx context.Context, address string, usedBytes int) error
	// get the current day's usage
	// read only when resuming an interrupted (errored) task
	CheckUsage(ctx context.Context, address string) (*Record, error)
}

type Record struct {
	Address   string `dynamodbav:"address"`
	BytesUsed int    `dynamodbav:"bytes_used"`
}

var ttl = time.Hour * 24 * 30

type DefaultUsageRepository struct {
	db        *dynamodb.Client
	tableName string
}

func NewUsageRepository(db *dynamodb.Client, tableName string) *DefaultUsageRepository {
	return &DefaultUsageRepository{
		db:        db,
		tableName: tableName,
	}
}

func (r DefaultUsageRepository) buildPK(address string) string {
	return fmt.Sprintf("IMAP_USAGE#%s", address)
}

const dateFormat = "2006-01-02"

func (r DefaultUsageRepository) buildSK() string {
	return time.Now().UTC().Format(dateFormat)
}

func (r DefaultUsageRepository) buildKey(address string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		mailboxsweep.DynamoPK: &types.AttributeValueMemberS{
			Value: r.buildPK(address),
		},
		mailboxsweep.DynamoSK: &types.AttributeValueMemberS{
			Value: r.buildSK(),
		},
	}
}

func (r DefaultUsageRepository) Update(ctx context.Context, address string, usedBytes int) error {
	key := r.buildKey(address)
	_, err := r.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        &r.tableName,
		Key:              key,
		UpdateExpression: aws.String("SET #bytes_used = :bytes_used, #address = if_not_exists(#address, :address), #expires = if_not_exists(#expires, :expires)"),
		ExpressionAttributeNames: map[string]string{
			"#bytes_used": "bytes_used",
			"#address":    "address",
			"#expires":    "expires_at",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":bytes_used": &types.AttributeValueMemberN{
				Value: strconv.Itoa(usedBytes),
			},
			":address": &types.AttributeValueMemberS{
				Value: address,
			},
			":expires": &types.AttributeValueMemberN{
				Value: strconv.FormatInt(
					time.Now().Add(ttl).Unix(),
					10,
				),
			},
		},
	})
	return err
}

func (r DefaultUsageRepository) CheckUsage(ctx context.Context, address string) (*Record, error) {
	key := r.buildKey(address)
	res, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key:       key,
		// this may be subject to frequent updates, use consistent reads
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	rec := Record{}
	err = attributevalue.UnmarshalMap(res.Item, &rec)
	return &rec, err
}
