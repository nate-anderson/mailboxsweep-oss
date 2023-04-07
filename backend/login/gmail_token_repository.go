package login

import (
	"context"
	"fmt"
	"mailboxsweep"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type GmailTokenRepository interface {
	Get(ctx context.Context, token string) (*GmailLoginToken, error)
	Store(ctx context.Context, t GmailLoginToken) error
}

type DynamoGmailTokenRepository struct {
	tableName string
	db        *dynamodb.Client
}

func NewDynamoGmailTokenRepository(db *dynamodb.Client, tableName string) *DynamoGmailTokenRepository {
	return &DynamoGmailTokenRepository{
		tableName: tableName,
		db:        db,
	}
}

func (r DynamoGmailTokenRepository) buildKey(token string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		mailboxsweep.DynamoPK: &types.AttributeValueMemberS{
			Value: r.buildPK(token),
		},
		mailboxsweep.DynamoSK: &types.AttributeValueMemberS{
			Value: "0",
		},
	}
}

func (r DynamoGmailTokenRepository) buildPK(token string) string {
	return fmt.Sprintf("GMAIL_TOKEN#%s", token)
}

type GmailLoginToken struct {
	Address string `dynamodbav:"address"`
	Token   string `dyamodbav:"token"`
	Cookie  string `dynamodbav:"cookie"`
	Expires int64  `dynamodbav:"expires_at"`
}

func (r DynamoGmailTokenRepository) Store(ctx context.Context, t GmailLoginToken) error {
	key := r.buildKey(t.Token)

	attrs, err := attributevalue.MarshalMap(t)
	if err != nil {
		return err
	}

	item := mailboxsweep.MergeAttributeValueMaps(key, attrs)

	_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      item,
	})

	return err
}

func (r DynamoGmailTokenRepository) Get(ctx context.Context, token string) (*GmailLoginToken, error) {
	key := r.buildKey(token)
	res, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key:       key,
	})
	if err != nil {
		return nil, err
	}

	if len(res.Item) == 0 {
		return nil, mailboxsweep.ErrNotFound
	}

	t := GmailLoginToken{}
	if err := attributevalue.UnmarshalMap(res.Item, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

var _ GmailTokenRepository = DynamoGmailTokenRepository{}
