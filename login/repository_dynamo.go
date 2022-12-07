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

func (r DynamoLinkRepository) formatGSI1PK(address string) string {
	return fmt.Sprintf("LOGIN_UUID#%s", address)
}

func (r DynamoLinkRepository) formatGSI1SK() string {
	return "0"
}

func (r DynamoLinkRepository) buildFullKey(token, uuid string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		mailboxsweep.DynamoPK: &types.AttributeValueMemberS{
			Value: r.formatPK(token),
		},
		mailboxsweep.DynamoSK: &types.AttributeValueMemberS{
			Value: r.formatSK(),
		},
		mailboxsweep.DynamoGSI1PK: &types.AttributeValueMemberS{
			Value: r.formatGSI1PK(uuid),
		},
		mailboxsweep.DynamoGSI1SK: &types.AttributeValueMemberS{
			Value: r.formatGSI1SK(),
		},
	}
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

// delete existing, use after login and when generating new keys
func (r DynamoLinkRepository) InvalidateByAddress(ctx context.Context, address string) error {
	findRes, err := r.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &r.tableName,
		IndexName:              aws.String(mailboxsweep.DynamoGSI1),
		KeyConditionExpression: aws.String("#gsi1pk = :gsi1pk AND #gsi1sk = :gsi1sk"),
		ExpressionAttributeNames: map[string]string{
			"#gsi1pk": mailboxsweep.DynamoGSI1PK,
			"#gsi1sk": mailboxsweep.DynamoGSI1SK,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":gsi1pk": &types.AttributeValueMemberS{
				Value: r.formatGSI1PK(address),
			},
			":gsi1sk": &types.AttributeValueMemberS{
				Value: r.formatGSI1SK(),
			},
		},
	})
	if err != nil {
		return err
	}

	// probably need to update this to only include the PK and SK of the GSI
	keysToDelete := make([]types.WriteRequest, findRes.Count)
	for i, item := range findRes.Items {
		keysToDelete[i] = types.WriteRequest{
			DeleteRequest: &types.DeleteRequest{
				Key: item,
			},
		}
	}

	if len(keysToDelete) > 0 {
		_, err = r.db.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				r.tableName: keysToDelete,
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
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
	err = attributevalue.UnmarshalMap(res.Item, &l)
	return &l, err
}
