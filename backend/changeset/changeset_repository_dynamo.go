package changeset

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"mailboxsweep"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoChangesetRepository struct {
	db *dynamodb.Client

	// use modulus of the hash to determine which page to put a path into
	pageIDHashMod uint64
	tableName     string
}

// structure used by dynamo
type changeSetDocument struct {
	ChangeSet

	// deletion_set#user@gmail.com (inbox)
	PK string `dynamodbav:"pk"`

	// deletion_set#FOLDER (folder)
	SK string `dynamodbav:"sk"`

	Created time.Time `dynamodbav:"created"`
	Updated time.Time `dynamodbav:"updated"`

	// unix timestamp (seconds)
	TTL int64 `dynamodbav:"expires_at"`
}

func NewDynamoChangesetRepository(
	db *dynamodb.Client,
	tableName string,
	pageIDHashMod uint64,
) (*DynamoChangesetRepository, error) {
	if pageIDHashMod == 0 {
		return nil, errors.New("pageIDHashMod must be > 0")
	}
	return &DynamoChangesetRepository{
		db:            db,
		tableName:     tableName,
		pageIDHashMod: pageIDHashMod,
	}, nil
}

func (r DynamoChangesetRepository) FindByInboxAndOperation(ctx context.Context, inbox string, operation UpdateOperation, consistentRead bool) (*ChangeSet, error) {
	items, err := r.findDynamoItemsByInboxAndOperation(ctx, inbox, operation, consistentRead)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, mailboxsweep.NotFoundError(errors.New("no changeset documents found for inbox"))
	}

	results := make([]changeSetDocument, len(items))
	err = attributevalue.UnmarshalListOfMaps(items, &results)
	if err != nil {
		return nil, err
	}

	cs := ChangeSet{
		Inbox:     inbox,
		Operation: operation,
		Paths:     []string{},
	}
	for _, doc := range results {
		cs.Paths = append(cs.Paths, doc.Paths...)
	}
	return &cs, nil
}

func (r DynamoChangesetRepository) findDynamoItemsByInboxAndOperation(ctx context.Context, inbox string, operation UpdateOperation, consistentRead bool) ([]map[string]types.AttributeValue, error) {
	res, err := r.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &r.tableName,
		KeyConditionExpression: aws.String("#pk = :email"),
		ExpressionAttributeNames: map[string]string{
			"#pk": mailboxsweep.DynamoPK,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":email": &types.AttributeValueMemberS{
				Value: r.formatPK(inbox, operation),
			},
		},
		ConsistentRead: &consistentRead,
	})
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return res.Items, nil
}

func (r DynamoChangesetRepository) formatPK(inbox string, operation UpdateOperation) string {
	return fmt.Sprintf("CHANGESET#%s#%s", inbox, operation)
}

func (r DynamoChangesetRepository) formatSK(partitionID uint64) string {
	return fmt.Sprintf("PAGE#%d", partitionID)
}

func (r DynamoChangesetRepository) formatLSI1(operation UpdateOperation) string {
	return fmt.Sprintf("OPERATION#%s", operation)
}

func (r DynamoChangesetRepository) partitionID(path string) (uint64, error) {
	hexVal := md5.Sum([]byte(path))
	hexStr := hex.EncodeToString(hexVal[:4])
	dec, err := strconv.ParseUint(hexStr, 16, 64)
	if err != nil {
		return 0, err
	}
	return dec % r.pageIDHashMod, nil
}

func (r DynamoChangesetRepository) partitionSet(changeSet ChangeSet) (map[string]ChangeSet, error) {
	partitionMap := make(map[uint64][]string)
	for _, path := range changeSet.Paths {
		partitionID, err := r.partitionID(path)
		if err != nil {
			return nil, err
		}
		partitionMap[partitionID] = append(partitionMap[partitionID], path)
	}

	res := make(map[string]ChangeSet)
	for partitionID, paths := range partitionMap {
		sk := r.formatSK(partitionID)
		c := changeSet
		c.Paths = paths
		res[sk] = c
	}
	return res, nil
}

// Append to a changeset, splitting it into pages using the hash function
func (r DynamoChangesetRepository) Append(ctx context.Context, changeSet ChangeSet) (int, error) {
	now := time.Now()
	expires := now.Add(mailboxsweep.DynamoUserDataTTL).Unix()

	partitionMap, err := r.partitionSet(changeSet)
	if err != nil {
		return 0, err
	}

	pk := r.formatPK(changeSet.Inbox, changeSet.Operation)

	// do this serially because
	// a) it requires less provisioned throughput, and
	// b) it allows using ReturnValues to calculate the diff
	// if performance is an issue, consider a Transaction
	// or doing this concurrently in an errgroup
	for sk, set := range partitionMap {
		_, err := r.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: &r.tableName,
			Key: map[string]types.AttributeValue{
				mailboxsweep.DynamoPK: &types.AttributeValueMemberS{
					Value: pk,
				},
				mailboxsweep.DynamoSK: &types.AttributeValueMemberS{
					Value: sk,
				},
			},
			UpdateExpression: aws.String(
				`SET
				#inbox = :inbox,
				#lsi1 = :lsi1,
				#operation = :operation,
				#created = if_not_exists(created, :created),
				#updated = :updated,
				#expires = :expires
				
				ADD #paths :paths`,
			),
			ExpressionAttributeNames: map[string]string{
				"#inbox":     "inbox",
				"#lsi1":      mailboxsweep.DynamoLSI1SK,
				"#operation": "operation",
				"#paths":     "paths",
				"#created":   "created",
				"#updated":   "updated",
				"#expires":   mailboxsweep.DynamoExpires,
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":inbox": &types.AttributeValueMemberS{
					Value: changeSet.Inbox,
				},
				":paths": &types.AttributeValueMemberSS{
					Value: set.Paths,
				},
				":created": &types.AttributeValueMemberS{
					Value: now.Format(mailboxsweep.DynamoTimeFormat),
				},
				":updated": &types.AttributeValueMemberS{
					Value: now.Format(mailboxsweep.DynamoTimeFormat),
				},
				":expires": &types.AttributeValueMemberN{
					Value: strconv.FormatInt(expires, 10),
				},
				":lsi1": &types.AttributeValueMemberS{
					Value: r.formatLSI1(changeSet.Operation),
				},
				":operation": &types.AttributeValueMemberS{
					Value: string(changeSet.Operation),
				},
			},
			// this function returns the number of paths added,
			// fetching the old ones allows this calculation
			ReturnValues: types.ReturnValueUpdatedOld,
		})
		if err != nil {
			return 0, fmt.Errorf("UpdateItem failed: %w", err)
		}
	}

	return len(changeSet.Paths), nil
}

func (r DynamoChangesetRepository) Remove(ctx context.Context, changeSet ChangeSet) (int, error) {
	now := time.Now()
	expires := now.Add(mailboxsweep.DynamoUserDataTTL).Unix()

	// track diff
	oldPaths := make([]string, 0, len(changeSet.Paths))

	partitionMap, err := r.partitionSet(changeSet)
	if err != nil {
		return 0, err
	}

	pk := r.formatPK(changeSet.Inbox, changeSet.Operation)

	// do this serially because
	// a) it requires less provisioned throughput, and
	// b) it allows using ReturnValues to calculate the diff
	// if performance is an issue, consider a Transaction
	// or doing this concurrently in an errgroup
	for sk, set := range partitionMap {
		res, err := r.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: &r.tableName,
			Key: map[string]types.AttributeValue{
				mailboxsweep.DynamoPK: &types.AttributeValueMemberS{
					Value: pk,
				},
				mailboxsweep.DynamoSK: &types.AttributeValueMemberS{
					Value: sk,
				},
			},
			ConditionExpression: aws.String("#pk = :pk AND #sk = :sk"),
			UpdateExpression: aws.String(
				`SET
				#updated = :updated,
				#expires = :expires
				
				DELETE #paths :paths`,
			),
			ExpressionAttributeNames: map[string]string{
				"#pk":      mailboxsweep.DynamoPK,
				"#sk":      mailboxsweep.DynamoSK,
				"#paths":   "paths",
				"#updated": "updated",
				"#expires": mailboxsweep.DynamoExpires,
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{
					Value: pk,
				},
				":sk": &types.AttributeValueMemberS{
					Value: sk,
				},
				":paths": &types.AttributeValueMemberSS{
					Value: set.Paths,
				},
				":updated": &types.AttributeValueMemberS{
					Value: now.Format(mailboxsweep.DynamoTimeFormat),
				},
				":expires": &types.AttributeValueMemberN{
					Value: strconv.FormatInt(expires, 10),
				},
			},
			// this function returns the number of paths added,
			// fetching the old ones allows this calculation
			ReturnValues: types.ReturnValueUpdatedOld,
		})
		if err != nil {
			return 0, fmt.Errorf("UpdateItem failed: %w", err)
		}

		oldList := []string{}
		if err = attributevalue.Unmarshal(res.Attributes["paths"], &oldList); err != nil {
			return 0, fmt.Errorf("failed to unmarshal pre-update values to calculate diff: %w", err)
		}
		oldPaths = append(oldPaths, oldList...)
	}

	presentInOld := make(map[string]bool)
	for _, old := range oldPaths {
		presentInOld[old] = true
	}
	diff := 0
	for _, removed := range changeSet.Paths {
		if _, ok := presentInOld[removed]; ok {
			diff--
		}
	}
	return diff, nil
}

func (r DynamoChangesetRepository) buildExecutedChangesetPK(inbox string, operation UpdateOperation, executedAt time.Time) string {
	return fmt.Sprintf("EXECUTED_CHANGESET#%s#%s#%d", inbox, operation, executedAt.Unix())
}

// move changeset to new primary keys so they are still in our records but no longer shown to users
func (r DynamoChangesetRepository) Archive(ctx context.Context, inbox string, operation UpdateOperation) error {
	items, err := r.findDynamoItemsByInboxAndOperation(ctx, inbox, operation, true)
	if err != nil {
		return fmt.Errorf("failed to find existing changeset to mark read: %w", err)
	}

	executedAt := time.Now()

	// do this serially for simplicity
	// it's slow, but it's the last step, the user should still be happy!
	for _, item := range items {
		// copy the data to the new key
		oldPK := item[mailboxsweep.DynamoPK]
		sk := item[mailboxsweep.DynamoSK]
		updatedKey := r.buildExecutedChangesetPK(inbox, operation, executedAt)
		item[mailboxsweep.DynamoPK] = &types.AttributeValueMemberS{
			Value: updatedKey,
		}

		_, err := r.db.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: &r.tableName,
			Item:      item,
		})
		if err != nil {
			return fmt.Errorf("failed to copy changeset to executed-version key: %w", err)
		}

		// delete the original
		_, err = r.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: &r.tableName,
			Key: map[string]types.AttributeValue{
				mailboxsweep.DynamoPK: oldPK,
				mailboxsweep.DynamoSK: sk,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to delete original changeset document %s: %w", sk, err)
		}
	}

	return nil
}
