package login

import (
	"context"
	"fmt"
	"mailboxsweep"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoLinkRepository struct {
	db        *dynamodb.Client
	tableName string
}

func NewDynamoLinkRepository(db *dynamodb.Client, table string) *DynamoLinkRepository {
	return &DynamoLinkRepository{
		db,
		table,
	}
}

func (r DynamoLinkRepository) Store(ctx context.Context, link Link) error {
	key := r.buildFullKey(link.Token, link.UserUUID)
	attrs, err := attributevalue.MarshalMap(link)
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

func (r DynamoLinkRepository) formatPK(token string) string {
	return fmt.Sprintf("LOGIN#%s", token)
}

func (r DynamoLinkRepository) formatSK() string {
	return "0"
}

func (r DynamoLinkRepository) formatGSI1PK(uuid string) string {
	return fmt.Sprintf("LOGIN_UUID#%s", uuid)
}

func (r DynamoLinkRepository) formatGSI1SK() string {
	return "0"
}

func (r DynamoLinkRepository) buildFullKey(token, userUUID string) map[string]types.AttributeValue {
	return mailboxsweep.MergeAttributeValueMaps(
		r.buildPrimaryKey(token),
		r.buildGSIKey(userUUID),
	)
}

func (r DynamoLinkRepository) buildPrimaryKey(token string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		mailboxsweep.DynamoPK: &types.AttributeValueMemberS{
			Value: r.formatPK(token),
		},
		mailboxsweep.DynamoSK: &types.AttributeValueMemberS{
			Value: r.formatSK(),
		},
	}
}

func (r DynamoLinkRepository) buildGSIKey(userUUID string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		mailboxsweep.DynamoGSI1PK: &types.AttributeValueMemberS{
			Value: r.formatGSI1PK(userUUID),
		},
		mailboxsweep.DynamoGSI1SK: &types.AttributeValueMemberS{
			Value: r.formatGSI1SK(),
		},
	}
}

// delete existing, use after login and when generating new keys
func (r DynamoLinkRepository) InvalidateByUserUUID(ctx context.Context, userUUID string) error {
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
				Value: r.formatGSI1PK(userUUID),
			},
			":gsi1sk": &types.AttributeValueMemberS{
				Value: r.formatGSI1SK(),
			},
		},
	})
	if err != nil {
		return err
	}

	if keyRes.Count < 1 {
		return nil
	}

	key := keyRes.Items[0]
	// delete GSI keys
	delete(key, mailboxsweep.DynamoGSI1PK)
	delete(key, mailboxsweep.DynamoGSI1SK)
	_, err = r.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &r.tableName,
		Key:       key,
	})
	return err
}

func (r DynamoLinkRepository) Find(ctx context.Context, token string) (*Link, error) {
	key := r.buildPrimaryKey(token)
	res, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key:       key,
	})
	if err != nil {
		return nil, err
	}

	l := Link{}
	if len(res.Item) == 0 {
		return nil, mailboxsweep.ErrNotFound
	}
	err = attributevalue.UnmarshalMap(res.Item, &l)
	return &l, err
}

func (r DynamoLinkRepository) FindByUserUUID(ctx context.Context, userUUID string) (*Link, error) {
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
				Value: r.formatGSI1PK(userUUID),
			},
			":gsi1sk": &types.AttributeValueMemberS{
				Value: r.formatGSI1SK(),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if keyRes.Count < 1 {
		return nil, mailboxsweep.ErrNotFound
	}

	key := keyRes.Items[0]
	delete(key, mailboxsweep.DynamoGSI1SK)
	delete(key, mailboxsweep.DynamoGSI1PK)

	itemRes, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key:       key,
	})
	if err != nil {
		return nil, err
	}

	link := Link{}
	if err := attributevalue.UnmarshalMap(itemRes.Item, &link); err != nil {
		return nil, err
	}
	return &link, nil
}
