package payment

import (
	"context"
	"errors"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/inboxuser"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type cart struct {
	// currently this is just a stripe checkout session ID
	ExternalID  string                  `dynamodbav:"external_id"`
	Method      inboxuser.PaymentMethod `dynamodbav:"payment_method"`
	UserAddress string                  `dynamodbav:"user_address"`
	Plan        inboxuser.Plan          `dynamodbav:"plan"`
	CreatedAt   time.Time               `dynamodbav:"created_at"`
	Expires     int64                   `dynamodbav:"expires_at"`
}

type CartRepository struct {
	db    *dynamodb.Client
	table string
}

func NewCartRepository(db *dynamodb.Client, table string) *CartRepository {
	return &CartRepository{
		db:    db,
		table: table,
	}
}

func (r CartRepository) buildKey(method inboxuser.PaymentMethod, externalID string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		mailboxsweep.DynamoPK: &types.AttributeValueMemberS{
			Value: r.formatPK(method, externalID),
		},
		mailboxsweep.DynamoSK: &types.AttributeValueMemberS{
			Value: r.formatSK(),
		},
	}
}

func (r CartRepository) formatPK(method inboxuser.PaymentMethod, externalID string) string {
	return fmt.Sprintf("PAYMENT#%s#%s", method, externalID)
}

func (r CartRepository) formatSK() string {
	return "0"
}

func (r CartRepository) store(
	ctx context.Context,
	c cart,
) error {
	key := r.buildKey(c.Method, c.ExternalID)

	attrs, err := attributevalue.MarshalMap(c)
	if err != nil {
		return err
	}
	item := mailboxsweep.MergeAttributeValueMaps(
		attrs,
		key,
	)

	_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.table,
		Item:      item,
	})
	return err
}

func (r CartRepository) find(
	ctx context.Context,
	method inboxuser.PaymentMethod,
	externalID string,
) (*cart, error) {
	key := r.buildKey(method, externalID)
	res, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{
		Key:       key,
		TableName: &r.table,
	})
	if err != nil {
		return nil, err
	}

	if len(res.Item) == 0 {
		err = errors.New("no cart found")
		return nil, mailboxsweep.NotFoundError(err)
	}

	var c cart
	err = attributevalue.UnmarshalMap(res.Item, &c)
	return &c, err
}
