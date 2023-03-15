package imapserver

import (
	"context"
	"errors"
	"fmt"
	"mailboxsweep"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DefaultRepository struct {
	db        *dynamodb.Client
	tableName string
}

func NewIMAPServerRepository(db *dynamodb.Client, tableName string) *DefaultRepository {
	return &DefaultRepository{
		db,
		tableName,
	}
}

func (r DefaultRepository) buildPK(domain string) string {
	return fmt.Sprintf("IMAP_SERVER#%s", domain)
}

func (r DefaultRepository) buildSK() string {
	return "0"
}

func (r DefaultRepository) buildGSI1(userAddress string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		mailboxsweep.DynamoGSI1PK: &types.AttributeValueMemberS{
			Value: r.buildGSI1PK(userAddress),
		},
		mailboxsweep.DynamoGSI1SK: &types.AttributeValueMemberS{
			Value: r.buildGSI1SK(),
		},
	}
}

func (r DefaultRepository) buildGSI1PK(userAddress string) string {
	return fmt.Sprintf("IMAPSERVER_USER#%s", userAddress)
}

func (r DefaultRepository) buildGSI1SK() string {
	return "0"
}

func (r DefaultRepository) buildKey(domain string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		mailboxsweep.DynamoPK: &types.AttributeValueMemberS{
			Value: r.buildPK(domain),
		},
		mailboxsweep.DynamoSK: &types.AttributeValueMemberS{
			Value: r.buildSK(),
		},
	}
}

func (r DefaultRepository) FindByDomain(ctx context.Context, domain string) (*Server, error) {
	key := r.buildKey(domain)
	res, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key:       key,
	})
	if err != nil {
		return nil, err
	}

	if len(res.Item) == 0 {
		return nil, mailboxsweep.NotFoundError(fmt.Errorf("domain %s not known", domain))
	}

	var s Server
	if err := attributevalue.UnmarshalMap(res.Item, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

func (r DefaultRepository) Store(ctx context.Context, server Server) error {
	key := r.buildKey(server.Domain)
	attrs, err := attributevalue.MarshalMap(server)
	if err != nil {
		return err
	}

	var item map[string]types.AttributeValue

	if server.CreatedBy == nil {
		item = mailboxsweep.MergeAttributeValueMaps(key, attrs)
	} else {
		gsi := r.buildGSI1(*server.CreatedBy)
		item = mailboxsweep.MergeAttributeValueMaps(key, gsi, attrs)
	}

	_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      item,
	})
	return err
}

func (r DefaultRepository) FindByUserAddress(ctx context.Context, userAddress string) (*Server, error) {
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
				Value: r.buildGSI1PK(userAddress),
			},
			":gsi1sk": &types.AttributeValueMemberS{
				Value: r.buildGSI1SK(),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(keyRes.Items) == 0 {
		return nil, mailboxsweep.NotFoundError(errors.New("no server details found for this user"))
	}

	key := keyRes.Items[0]
	// delete GSI keys
	delete(key, mailboxsweep.DynamoGSI1PK)
	delete(key, mailboxsweep.DynamoGSI1SK)

	if err != nil {
		return nil, err
	}

	itemRes, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key:       key,
	})
	if err != nil {
		return nil, err
	}

	s := Server{}
	err = attributevalue.UnmarshalMap(itemRes.Item, &s)
	return &s, err
}
