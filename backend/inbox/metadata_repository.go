package inbox

import (
	"context"
	"errors"
	"fmt"
	"mailboxsweep"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// allow caching some metadata so we don't have to make IMAP
// requests for simple page views
// these cache entries should have a pretty short TTL because
// this stuff can change

type Metadata struct {
	Inbox        string   `dynamodbav:"inbox"`
	Folders      []string `dynamodbav:"folders"`
	MessageCount int      `dynamodbav:"message_count"`
	UnreadCount  int      `dynamodbav:"unread_count"`
	Expires      int64    `dynamodbav:"expires_at"`
}

type MetadataRepository interface {
	Store(ctx context.Context, metadata Metadata) error
	Get(ctx context.Context, address string) (*Metadata, error)
	Delete(ctx context.Context, address string) error
}

type DefaultMetadataRepository struct {
	db        *dynamodb.Client
	tableName string
}

func NewMetadataRepository(db *dynamodb.Client, table string) *DefaultMetadataRepository {
	return &DefaultMetadataRepository{
		db,
		table,
	}
}

func (r DefaultMetadataRepository) buildPK(inbox string) string {
	return fmt.Sprintf("INBOX_METADATA#%s", inbox)
}

func (r DefaultMetadataRepository) buildSK() string {
	return "0"
}

func (r DefaultMetadataRepository) buildKey(inbox string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		mailboxsweep.DynamoPK: &types.AttributeValueMemberS{
			Value: r.buildPK(inbox),
		},
		mailboxsweep.DynamoSK: &types.AttributeValueMemberS{
			Value: r.buildSK(),
		},
	}
}

func (r DefaultMetadataRepository) Store(ctx context.Context, metadata Metadata) error {
	key := r.buildKey(metadata.Inbox)
	attributes, err := attributevalue.MarshalMap(metadata)
	if err != nil {
		return err
	}
	item := mailboxsweep.MergeAttributeValueMaps(key, attributes)
	_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      item,
	})
	return err
}

func (r DefaultMetadataRepository) Get(ctx context.Context, address string) (*Metadata, error) {
	key := r.buildKey(address)
	res, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key:       key,
	})
	if err != nil {
		return nil, err
	}

	if len(res.Item) == 0 {
		return nil, mailboxsweep.NotFoundError(errors.New("no metadata found"))
	}

	m := Metadata{}
	err = attributevalue.UnmarshalMap(res.Item, &m)
	return &m, err
}

func (r DefaultMetadataRepository) Delete(ctx context.Context, address string) error {
	key := r.buildKey(address)
	res, err := r.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		Key:          key,
		TableName:    &r.tableName,
		ReturnValues: types.ReturnValueAllOld,
	})
	if err != nil {
		return err
	}

	if len(res.Attributes) == 0 {
		err = errors.New("no metadata stored for that address")
		return mailboxsweep.NotFoundError(err)
	}

	return nil
}
