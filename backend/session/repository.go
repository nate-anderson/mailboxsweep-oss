package session

import (
	"context"
	"errors"
	"fmt"
	"mailboxsweep"
	"net/http"
	"strings"
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
	ListByUser(ctx context.Context, address string) (cookieTokens []string, err error)
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

func (r DynamoSessionRepository) buildGSI1PK(address string) string {
	return fmt.Sprintf("USER_SESSION#%s", address)
}

func (r DynamoSessionRepository) buildGSI1(address, token string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		mailboxsweep.DynamoGSI1PK: &types.AttributeValueMemberS{
			Value: r.buildGSI1PK(address),
		},
		mailboxsweep.DynamoGSI1SK: &types.AttributeValueMemberS{
			Value: token,
		},
	}
}

func (r DynamoSessionRepository) Store(ctx context.Context, sess Session) error {
	itemAttrs, err := attributevalue.MarshalMap(sess)
	if err != nil {
		return err
	}
	key := r.buildPK(sess.Token)
	gsi1 := r.buildGSI1(sess.UserAddress, sess.Token)
	item := mailboxsweep.MergeAttributeValueMaps(itemAttrs, key, gsi1)

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
		return nil, mailboxsweep.NotFoundError(errors.New("session not found"))
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

// avoid having to fetch full items when we have looked up keys for a user's active sessions
func (r DynamoSessionRepository) parsePK(pk string) (string, error) {
	toks := strings.Split(pk, "#")
	if len(toks) != 2 {
		return "", errors.New("encountered malformed PK for session key")
	}
	return toks[1], nil
}

func (r DynamoSessionRepository) ListByUser(ctx context.Context, address string) ([]string, error) {
	var (
		sessionTokens []string
		cursor        map[string]types.AttributeValue
	)

	for {
		res, err := r.db.Query(ctx, &dynamodb.QueryInput{
			TableName:         &r.tableName,
			ExclusiveStartKey: cursor,
			ExpressionAttributeNames: map[string]string{
				"#GSI1PK": mailboxsweep.DynamoGSI1PK,
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":address": &types.AttributeValueMemberS{
					Value: r.buildGSI1PK(address),
				},
			},
			KeyConditionExpression: aws.String("#GSI1PK = :address"),
			IndexName:              &mailboxsweep.DynamoGSI1,
		})

		if err != nil {
			return nil, err
		}

		if len(res.Items) == 0 {
			return nil, nil
		}

		for _, item := range res.Items {
			if stringVal, ok := item[mailboxsweep.DynamoPK].(*types.AttributeValueMemberS); ok {
				token, err := r.parsePK(stringVal.Value)
				if err != nil {
					return nil, errors.New("invalid key value from session query result")
				}
				sessionTokens = append(sessionTokens, token)
			} else {
				return nil, errors.New("got unexpected type of attribute in session query results")
			}
		}

		if len(res.LastEvaluatedKey) > 0 {
			cursor = res.LastEvaluatedKey
		} else {
			break
		}
	}

	return sessionTokens, nil
}

func (s Session) AuthCookie(isDev bool, domain string) *http.Cookie {
	cookie := &http.Cookie{
		Name:     SessionAuthCookie,
		Value:    s.Token,
		Expires:  time.Unix(s.Expires, 0),
		Path:     "/",
		HttpOnly: true,
		Secure:   !isDev,
		Domain:   domain,
		SameSite: http.SameSiteStrictMode,
	}
	return cookie
}

const isLoggedInCookie = "X-MailboxSweep-LoggedIn"

func (s Session) IsLoggedInCookie(domain string) *http.Cookie {
	return &http.Cookie{
		Name:    isLoggedInCookie,
		Value:   "true",
		Expires: time.Unix(s.Expires, 0),
		Path:    "/",
		// allow the client to see this cookie so it knows its authenticated
		HttpOnly: false,
		Secure:   true,
		Domain:   domain,
		SameSite: http.SameSiteStrictMode,
	}
}
