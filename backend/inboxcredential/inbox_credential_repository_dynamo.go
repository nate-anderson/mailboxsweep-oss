package inboxcredential

import (
	"context"
	"errors"
	"fmt"
	"mailboxsweep"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoInboxCredentialRepository struct {
	db        *dynamodb.Client
	tableName string
}

func NewDynamoInboxCredentialRepository(db *dynamodb.Client, table string) *DynamoInboxCredentialRepository {
	return &DynamoInboxCredentialRepository{
		db:        db,
		tableName: table,
	}
}

func (r DynamoInboxCredentialRepository) formatPK(address string) string {
	return fmt.Sprintf("CREDENTIAL#%s", address)
}

func (r DynamoInboxCredentialRepository) formatSK() string {
	return "0"
}

func (r DynamoInboxCredentialRepository) buildKey(address string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		mailboxsweep.DynamoPK: &types.AttributeValueMemberS{
			Value: r.formatPK(address),
		},
		mailboxsweep.DynamoSK: &types.AttributeValueMemberS{
			Value: r.formatSK(),
		},
	}
}

func (r DynamoInboxCredentialRepository) Upsert(ctx context.Context, credential Credential) error {
	attrs, err := attributevalue.MarshalMap(credential)
	if err != nil {
		return err
	}

	item := mailboxsweep.MergeAttributeValueMaps(
		attrs,
		r.buildKey(credential.Address()),
	)

	_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      item,
	})

	return err
}

func (r DynamoInboxCredentialRepository) FindByAddress(ctx context.Context, address string) (Credential, error) {
	key := r.buildKey(address)
	res, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key:       key,
	})
	if err != nil {
		return nil, err
	}

	if res.Item == nil {
		return nil, mailboxsweep.ErrNotFound
	}

	if _, ok := res.Item["is_imap"]; ok {
		ic := &IMAPInboxCredential{}
		if err := attributevalue.UnmarshalMap(res.Item, &ic); err != nil {
			return ic, fmt.Errorf("failed to unmarshal credential result: %w", err)
		}

		return ic, nil
	} else if _, ok := res.Item["is_gmail"]; ok {
		ic := &GmailInboxCredential{}
		if err := attributevalue.UnmarshalMap(res.Item, &ic); err != nil {
			return ic, fmt.Errorf("failed to unmarshal credential result: %w", err)
		}

		return ic, nil
	}

	return nil, errors.New("could not determine type of credential ")
}

func (r DynamoInboxCredentialRepository) DeleteByAddress(ctx context.Context, address string) error {
	_, err := r.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &r.tableName,
		Key:       r.buildKey(address),
	})
	return err
}
