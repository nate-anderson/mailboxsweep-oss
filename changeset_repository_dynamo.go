package mailboxsweep

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// structure used by dynamo
type changeSetDocument struct {
	ChangeSet

	// deletion_set#user@gmail.com (inbox)
	PK string `dynamodbav:"pk"`

	// deletion_set#INBOX (folder)
	SK string `dynamodbav:"folder"`

	Created time.Time `dynamodbav:"created"`
	Updated time.Time `dynamodbav:"updated"`

	// unix timestamp (seconds)
	TTL int64 `dynamodbav:"expire_at"`
}

type DynamoChangesetRepository struct {
	db        *dynamodb.Client
	tableName string
}

func NewDynamoChangesetRepository(db *dynamodb.Client, tableName string) *DynamoChangesetRepository {
	return &DynamoChangesetRepository{
		db:        db,
		tableName: tableName,
	}
}

func (d DynamoChangesetRepository) ListByInbox(ctx context.Context, inbox string) ([]ChangeSet, error) {
	res, err := d.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &d.tableName,
		KeyConditionExpression: ptr("#pk = :email"),
		ExpressionAttributeNames: map[string]string{
			"#pk": DynamoPK,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":email": &types.AttributeValueMemberS{
				Value: d.formatPK(inbox),
			},
		},
	})
	if err != nil {
		return []ChangeSet{}, fmt.Errorf("query failed: %w", err)
	}

	results := make([]changeSetDocument, len(res.Items))
	err = attributevalue.UnmarshalListOfMaps(res.Items, &results)
	if err != nil {
		return []ChangeSet{}, err
	}

	changeSets := make([]ChangeSet, len(results))
	for i, c := range results {
		changeSets[i] = c.ChangeSet
	}
	return changeSets, nil
}

func (d DynamoChangesetRepository) formatPK(inbox string) string {
	return "INBOX#" + strings.TrimSpace(inbox)
}

func (d DynamoChangesetRepository) formatSK(folder string) string {
	return "FOLDER#" + strings.TrimSpace(folder)
}

func (d DynamoChangesetRepository) buildKey(inbox, folder string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		DynamoPK: &types.AttributeValueMemberS{
			Value: d.formatPK(inbox),
		},
		DynamoSK: &types.AttributeValueMemberS{
			Value: d.formatSK(folder),
		},
	}
}

func (d DynamoChangesetRepository) Upsert(ctx context.Context, changeSet ChangeSet) (int, error) {
	uids := make([]string, len(changeSet.UIDs))
	for i, u := range changeSet.UIDs {
		uids[i] = fmt.Sprint(u)
	}

	now := time.Now()
	expires := now.Add(dynamoUserDataTTL).Unix()
	res, err := d.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &d.tableName,
		Key:       d.buildKey(changeSet.Inbox, changeSet.Folder),
		UpdateExpression: ptr(`SET
			#inbox = :inbox,
			#folder = :folder,
			#uids = :uids,
			#created = if_not_exists(created, :created),
			#updated = :updated,
			#expires = :expires`),
		ExpressionAttributeNames: map[string]string{
			"#inbox":   "inbox",
			"#folder":  "folder",
			"#uids":    "uids",
			"#created": "created",
			"#updated": "updated",
			"#expires": DynamoExpires,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inbox": &types.AttributeValueMemberS{
				Value: changeSet.Inbox,
			},
			":folder": &types.AttributeValueMemberS{
				Value: changeSet.Folder,
			},
			":uids": &types.AttributeValueMemberSS{
				Value: uids,
			},
			":created": &types.AttributeValueMemberS{
				Value: now.Format(dynamoTimeFormat),
			},
			":updated": &types.AttributeValueMemberS{
				Value: now.Format(dynamoTimeFormat),
			},
			":expires": &types.AttributeValueMemberN{
				Value: strconv.FormatInt(expires, 10),
			},
		},
		// this function returns the number of UIDs added,
		// fetching the old ones allows this calculation
		ReturnValues: types.ReturnValueAllOld,
	})
	if err != nil {
		return 0, fmt.Errorf("UpdateItem failed: %w", err)
	}

	oldList := []string{}
	if err = attributevalue.Unmarshal(res.Attributes["uids"], &oldList); err != nil {
		return 0, fmt.Errorf("failed to unmarshal pre-update values to calculate diff: %w", err)
	}
	diff := len(changeSet.UIDs) - len(oldList)
	return diff, nil
}
