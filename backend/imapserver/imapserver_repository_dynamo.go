package imapserver

import (
	"context"
	"fmt"
	"mailboxsweep"

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
		return nil, mailboxsweep.ErrNotFound
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

	item := mailboxsweep.MergeAttributeValueMaps(key, attrs)

	_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      item,
	})
	return err
}
