package inboxuser

import (
	"context"
	"fmt"
	"mailboxsweep"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoInboxUserRepository struct {
	db        *dynamodb.Client
	tableName string
}

func NewDynamoInboxUserRepository(db *dynamodb.Client, tableName string) *DynamoInboxUserRepository {
	return &DynamoInboxUserRepository{
		db,
		tableName,
	}
}

func (r DynamoInboxUserRepository) formatPK(address string) string {
	return fmt.Sprintf("USER#%s", address)
}

func (r DynamoInboxUserRepository) formatSK() string {
	return "0"
}

func (r DynamoInboxUserRepository) formatGSI1PK(userUUID string) string {
	return fmt.Sprintf("USER_UUID#%s", userUUID)
}

func (r DynamoInboxUserRepository) formatGSI1SK() string {
	return "0"
}

func (r DynamoInboxUserRepository) buildKey(address string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		mailboxsweep.DynamoPK: &types.AttributeValueMemberS{
			Value: r.formatPK(address),
		},
		mailboxsweep.DynamoSK: &types.AttributeValueMemberS{
			Value: r.formatSK(),
		},
	}
}

func (r DynamoInboxUserRepository) buildGSI1(userUUID string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		mailboxsweep.DynamoGSI1PK: &types.AttributeValueMemberS{
			Value: r.formatGSI1PK(userUUID),
		},
		mailboxsweep.DynamoGSI1SK: &types.AttributeValueMemberS{
			Value: r.formatGSI1SK(),
		},
	}
}

func (r DynamoInboxUserRepository) Store(ctx context.Context, user InboxUser) (string, error) {
	key := r.buildKey(user.Address)
	gsi1 := r.buildGSI1(user.UUID)
	attrs, err := attributevalue.MarshalMap(user)
	if err != nil {
		return "", fmt.Errorf("failed to marshal inbox user: %w", err)
	}
	item := mailboxsweep.MergeAttributeValueMaps(
		key,
		gsi1,
		attrs,
	)
	_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      item,
	})
	if err != nil {
		return "", err
	}
	return user.UUID, nil
}

func (r DynamoInboxUserRepository) FindByAddress(ctx context.Context, address string) (*InboxUser, error) {
	key := r.buildKey(address)
	res, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key:       key,
	})
	if err != nil {
		return nil, err
	}

	iu := InboxUser{}
	if err := attributevalue.UnmarshalMap(res.Item, &iu); err != nil {
		return nil, err
	}
	return &iu, nil
}

func (r DynamoInboxUserRepository) FindByUUID(ctx context.Context, uuid string) (*InboxUser, error) {
	gsi1pk := r.formatGSI1PK(uuid)
	gsi1sk := r.formatGSI1SK()
	keyRes, err := r.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &r.tableName,
		IndexName:              &mailboxsweep.DynamoGSI1,
		KeyConditionExpression: aws.String("#gsi1pk = :gsi1pk AND #gsi1sk = :gsi1sk"),
		ExpressionAttributeNames: map[string]string{
			"#gsi1pk": mailboxsweep.DynamoGSI1PK,
			"#gsi1sk": mailboxsweep.DynamoGSI1SK,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":gsi1pk": &types.AttributeValueMemberS{
				Value: gsi1pk,
			},
			":gsi1sk": &types.AttributeValueMemberS{
				Value: gsi1sk,
			},
		},
	})

	if err != nil {
		return nil, err
	}

	if keyRes.Count != 1 {
		return nil, fmt.Errorf("expected Query to return 1 user, got %d", keyRes.Count)
	}

	key := keyRes.Items[0]
	delete(key, mailboxsweep.DynamoGSI1PK)
	delete(key, mailboxsweep.DynamoGSI1SK)

	res, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key:       key,
	})
	if err != nil {
		return nil, err
	}

	user := InboxUser{}
	if err := attributevalue.UnmarshalMap(res.Item, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

func (r DynamoInboxUserRepository) Consent(ctx context.Context, address string, agreement Agreement) error {
	key := r.buildKey(address)
	_, err := r.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        &r.tableName,
		Key:              key,
		UpdateExpression: aws.String("SET #agreements.#agreement_code = :agreed_at"),
		ExpressionAttributeNames: map[string]string{
			"#agreements":     "agreements",
			"#agreement_code": string(agreement),
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":agreed_at": &types.AttributeValueMemberS{
				Value: time.Now().Format(mailboxsweep.DynamoTimeFormat),
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (r DynamoInboxUserRepository) StorePayment(ctx context.Context, address string, payment Payment) error {
	key := r.buildKey(address)
	paymentItem, err := attributevalue.Marshal(payment)
	if err != nil {
		return err
	}

	_, err = r.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        &r.tableName,
		Key:              key,
		UpdateExpression: aws.String("SET #payments = list_append(#payments, :payment)"),
		ExpressionAttributeNames: map[string]string{
			"#payments": "payments",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":payment": &types.AttributeValueMemberL{
				Value: []types.AttributeValue{
					paymentItem,
				},
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (r DynamoInboxUserRepository) TouchLastSeen(ctx context.Context, address string) error {
	key := r.buildKey(address)
	_, err := r.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		Key:              key,
		TableName:        &r.tableName,
		UpdateExpression: aws.String("SET #last_seen = :last_seen"),
		ExpressionAttributeNames: map[string]string{
			"#last_seen": "last_seen",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":last_seen": &types.AttributeValueMemberS{
				Value: time.Now().Format(mailboxsweep.DynamoTimeFormat),
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}
