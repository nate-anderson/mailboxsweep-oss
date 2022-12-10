package session

import (
	"context"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/inboxuser"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const sessionCookie = "X-MailboxSweep-Session"

type Repository interface {
	Store(ctx context.Context, sess session) (err error)
	Get(ctx context.Context, cookieToken string) (*session, error)
	Invalidate(ctx context.Context, cookieToken string) error
}

type authKeyT int

const authKey authKeyT = 1

type session struct {
	UserUUID string `dynamodbav:"user_uuid"`
	Token    string `dynamodbav:"token"`
	Expires  int64  `dynamodbav:"expires_at"`
}

func authorizeContext(ctx context.Context, user inboxuser.InboxUser) context.Context {
	r := context.WithValue(ctx, authKey, user.Address)
	return r
}

func NewDynamoSessionRepository(db *dynamodb.Client, tableName string) *DynamoSessionRepository {
	return &DynamoSessionRepository{
		db,
		tableName,
	}
}

type DynamoSessionRepository struct {
	db        *dynamodb.Client
	tableName string
}

func (r DynamoSessionRepository) formatPK(token string) string {
	return fmt.Sprintf("SESSION#%s", token)
}

func (r DynamoSessionRepository) formatSK() string {
	return "0"
}

func (r DynamoSessionRepository) buildPK(token string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		mailboxsweep.DynamoPK: &types.AttributeValueMemberS{
			Value: r.formatPK(token),
		},
		mailboxsweep.DynamoSK: &types.AttributeValueMemberS{
			Value: r.formatSK(),
		},
	}
}

func (r DynamoSessionRepository) Store(ctx context.Context, sess session) error {
	itemAttrs, err := attributevalue.MarshalMap(sess)
	if err != nil {
		return err
	}
	key := r.buildPK(sess.Token)
	item := mailboxsweep.MergeAttributeValueMaps(itemAttrs, key)

	_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      item,
	})
	if err != nil {
		return err
	}

	return err
}

func (r DynamoSessionRepository) Get(ctx context.Context, cookieToken string) (*session, error) {
	key := r.buildPK(cookieToken)
	itemRes, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      &r.tableName,
		Key:            key,
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	if len(itemRes.Item) == 0 {
		return nil, mailboxsweep.ErrNotFound
	}

	item := itemRes.Item
	sess := session{}
	err = attributevalue.UnmarshalMap(item, &sess)
	return &sess, err
}

func (r DynamoSessionRepository) Invalidate(ctx context.Context, cookieToken string) error {
	key := r.buildPK(cookieToken)
	_, err := r.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &r.tableName,
		Key:       key,
	})
	return err
}
