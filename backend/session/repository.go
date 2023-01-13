package session

import (
	"context"
	"fmt"
	"mailboxsweep"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Repository interface {
	Store(ctx context.Context, sess Session) (err error)
	Get(ctx context.Context, cookieToken string) (*Session, error)
	Invalidate(ctx context.Context, cookieToken string) error
}

type Session struct {
	UserAddress string `dynamodbav:"user_address"`
	Token       string `dynamodbav:"token"`
	Expires     int64  `dynamodbav:"expires_at"`
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

func (r DynamoSessionRepository) Store(ctx context.Context, sess Session) error {
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

func (r DynamoSessionRepository) Get(ctx context.Context, cookieToken string) (*Session, error) {
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
		return nil, mailboxsweep.NotFoundError(nil)
	}

	item := itemRes.Item
	sess := Session{}
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

func (s Session) AuthCookie(isDev bool) *http.Cookie {
	cookie := &http.Cookie{
		Name:     sessionAuthCookie,
		Value:    s.Token,
		Expires:  time.Unix(s.Expires, 0),
		Path:     "/",
		HttpOnly: true,
		Secure:   !isDev,
	}
	return cookie
}

const isLoggedInCookie = "X-MailboxSweep-LoggedIn"

func (s Session) IsLoggedInCookie() *http.Cookie {
	return &http.Cookie{
		Name:    isLoggedInCookie,
		Value:   "true",
		Expires: time.Unix(s.Expires, 0),
		Path:    "/",
		// allow the client to see this cookie so it knows its authenticated
		HttpOnly: false,
		Secure:   true,
	}
}
