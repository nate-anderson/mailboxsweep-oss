package mailboxsweep

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoChangesetRepository struct {
	db *dynamodb.Client

	// how many UIDs before we split into a new document?
	pageSize  int
	tableName string

	// used for breaking update sets into pages
	hashFunc uidHashFunc
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
	TTL int64 `dynamodbav:"expire_at"`

	lsi1 string `dynamodbav:"lsi1sk"`
}

func NewDynamoChangesetRepository(
	db *dynamodb.Client,
	tableName string,
	uidsPerDocument int,
	hashFunc uidHashFunc,
) (*DynamoChangesetRepository, error) {
	if uidsPerDocument <= 0 {
		return nil, errors.New("uidsPerDocument must be > 0")
	}
	return &DynamoChangesetRepository{
		db:        db,
		tableName: tableName,
		pageSize:  uidsPerDocument,
		hashFunc:  hashFunc,
	}, nil
}

func (d DynamoChangesetRepository) ListByInbox(ctx context.Context, inbox string, consistentRead bool) ([]ChangeSet, error) {
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
		ConsistentRead: &consistentRead,
	})
	if err != nil {
		return []ChangeSet{}, fmt.Errorf("query failed: %w", err)
	}

	results := make([]changeSetDocument, len(res.Items))
	err = attributevalue.UnmarshalListOfMaps(res.Items, &results)
	if err != nil {
		return []ChangeSet{}, err
	}

	// group changesets by folder
	// log.Fatalf("RESULTS: %+v", results)
	changeSetMap := make(map[string]ChangeSet, len(results))
	for _, c := range results {
		if existing, ok := changeSetMap[c.Folder]; !ok {
			changeSetMap[c.Folder] = c.ChangeSet
		} else {
			existing.UIDs = append(existing.UIDs, c.UIDs...)
			changeSetMap[c.Folder] = existing
		}
	}
	changeSets := make([]ChangeSet, 0, len(changeSetMap))
	for _, value := range changeSetMap {
		changeSets = append(changeSets, value)
	}
	return changeSets, nil
}

func (d DynamoChangesetRepository) formatPK(inbox string) string {
	return "CHANGESET#" + strings.TrimSpace(inbox)
}

func (d DynamoChangesetRepository) formatSK(folder string, operation UpdateOperation, partitionID uint64) string {
	return fmt.Sprintf("FOLDER#%s#%d#%s", strings.TrimSpace(folder), partitionID, operation)
}

func (d DynamoChangesetRepository) formatLSI1(folder string) string {
	return folder
}

func (d DynamoChangesetRepository) partitionID(uid string) (uint64, error) {
	p, err := d.hashFunc(uid)
	if err != nil {
		return 0, err
	}

	min := p / uint64(d.pageSize)
	return min, nil
}

func (d DynamoChangesetRepository) partitionSet(changeSet ChangeSet) (map[string]ChangeSet, error) {
	partitionMap := make(map[uint64][]string)
	for _, uid := range changeSet.UIDs {
		partitionID, err := d.partitionID(uid)
		if err != nil {
			return nil, err
		}
		partitionMap[partitionID] = append(partitionMap[partitionID], uid)
	}

	res := make(map[string]ChangeSet)
	for partitionID, uids := range partitionMap {
		sk := d.formatSK(changeSet.Folder, changeSet.Operation, partitionID)
		c := changeSet
		c.UIDs = uids
		res[sk] = c
	}
	return res, nil
}

// Append to a changeset, splitting it into pages if it contains more UIDs than the configured
// uidsPerDocument value
func (d DynamoChangesetRepository) Append(ctx context.Context, changeSet ChangeSet) (int, error) {
	now := time.Now()
	expires := now.Add(dynamoUserDataTTL).Unix()

	partitionMap, err := d.partitionSet(changeSet)
	if err != nil {
		return 0, err
	}

	pk := d.formatPK(changeSet.Inbox)

	// do this serially because
	// a) it requires less provisioned throughput, and
	// b) it allows using ReturnValues to calculate the diff
	// if performance is an issue, consider a Transaction
	// or doing this concurrently in an errgroup
	for sk, set := range partitionMap {
		_, err := d.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: &d.tableName,
			Key: map[string]types.AttributeValue{
				DynamoPK: &types.AttributeValueMemberS{
					Value: pk,
				},
				DynamoSK: &types.AttributeValueMemberS{
					Value: sk,
				},
			},
			UpdateExpression: ptr(
				`SET
				#inbox = :inbox,
				#lsi1 = :lsi1,
				#folder = :folder,
				#operation = :operation,
				#created = if_not_exists(created, :created),
				#updated = :updated,
				#expires = :expires
				
				ADD #uids :uids`,
			),
			ExpressionAttributeNames: map[string]string{
				"#inbox":     "inbox",
				"#lsi1":      DynamoLSI1SK,
				"#folder":    "folder",
				"#operation": "operation",
				"#uids":      "uids",
				"#created":   "created",
				"#updated":   "updated",
				"#expires":   DynamoExpires,
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":inbox": &types.AttributeValueMemberS{
					Value: changeSet.Inbox,
				},
				":folder": &types.AttributeValueMemberS{
					Value: changeSet.Folder,
				},
				":uids": &types.AttributeValueMemberSS{
					Value: set.UIDs,
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
				":lsi1": &types.AttributeValueMemberS{
					Value: d.formatLSI1(changeSet.Folder),
				},
				":operation": &types.AttributeValueMemberS{
					Value: string(changeSet.Operation),
				},
			},
			// this function returns the number of UIDs added,
			// fetching the old ones allows this calculation
			ReturnValues: types.ReturnValueUpdatedOld,
		})
		if err != nil {
			return 0, fmt.Errorf("UpdateItem failed: %w", err)
		}
	}

	return len(changeSet.UIDs), nil
}

func (d DynamoChangesetRepository) Remove(ctx context.Context, changeSet ChangeSet) (int, error) {
	now := time.Now()
	expires := now.Add(dynamoUserDataTTL).Unix()

	// track diff
	oldUIDs := make([]string, 0, len(changeSet.UIDs))

	partitionMap, err := d.partitionSet(changeSet)
	if err != nil {
		return 0, err
	}

	pk := d.formatPK(changeSet.Inbox)

	// do this serially because
	// a) it requires less provisioned throughput, and
	// b) it allows using ReturnValues to calculate the diff
	// if performance is an issue, consider a Transaction
	// or doing this concurrently in an errgroup
	for sk, set := range partitionMap {
		res, err := d.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: &d.tableName,
			Key: map[string]types.AttributeValue{
				DynamoPK: &types.AttributeValueMemberS{
					Value: pk,
				},
				DynamoSK: &types.AttributeValueMemberS{
					Value: sk,
				},
			},
			ConditionExpression: ptr("#pk = :pk AND #sk = :sk"),
			UpdateExpression: ptr(
				`SET
				#updated = :updated,
				#expires = :expires
				
				DELETE #uids :uids`,
			),
			ExpressionAttributeNames: map[string]string{
				"#pk":      DynamoPK,
				"#sk":      DynamoSK,
				"#uids":    "uids",
				"#updated": "updated",
				"#expires": DynamoExpires,
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{
					Value: pk,
				},
				":sk": &types.AttributeValueMemberS{
					Value: sk,
				},
				":uids": &types.AttributeValueMemberSS{
					Value: set.UIDs,
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
			ReturnValues: types.ReturnValueUpdatedOld,
		})
		if err != nil {
			return 0, fmt.Errorf("UpdateItem failed: %w", err)
		}

		oldList := []string{}
		if err = attributevalue.Unmarshal(res.Attributes["uids"], &oldList); err != nil {
			return 0, fmt.Errorf("failed to unmarshal pre-update values to calculate diff: %w", err)
		}
		oldUIDs = append(oldUIDs, oldList...)
	}

	presentInOld := make(map[string]bool)
	for _, old := range oldUIDs {
		presentInOld[old] = true
	}
	diff := 0
	for _, removed := range changeSet.UIDs {
		if _, ok := presentInOld[removed]; ok {
			diff++
		}
	}
	return diff, nil
}

func (d DynamoChangesetRepository) FindByInboxFolder(ctx context.Context, inbox, folder string, consistentRead bool) (map[UpdateOperation]ChangeSet, error) {
	keysRes, err := d.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &d.tableName,
		ConsistentRead:         &consistentRead,
		KeyConditionExpression: ptr("#pk = :pk AND #lsi1 = :lsi1"),
		ExpressionAttributeNames: map[string]string{
			"#pk":   DynamoPK,
			"#lsi1": DynamoLSI1SK,
		},
		IndexName: &DynamoLSI1,
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{
				Value: d.formatPK(inbox),
			},
			":lsi1": &types.AttributeValueMemberS{
				Value: d.formatLSI1(folder),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if keysRes.Count < 1 {
		return nil, fmt.Errorf("changeset for inbox %s and folder %s not found: %w", inbox, folder, err)
	}

	keyItems := []changeSetDocument{}
	if err = attributevalue.UnmarshalListOfMaps(keysRes.Items, &keyItems); err != nil {
		return nil, fmt.Errorf("failed to get gets for folder items: %w", err)
	}

	keys := make([]map[string]types.AttributeValue, len(keyItems))
	for i, item := range keyItems {
		keys[i] = map[string]types.AttributeValue{
			DynamoPK: &types.AttributeValueMemberS{
				Value: item.PK,
			},
			DynamoSK: &types.AttributeValueMemberS{
				Value: item.SK,
			},
		}
	}

	itemsRes, err := d.db.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			d.tableName: {
				Keys: keys,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve items with batch call: %w", err)
	}

	items := []changeSetDocument{}
	if err = attributevalue.UnmarshalListOfMaps(itemsRes.Responses[d.tableName], &items); err != nil {
		return nil, fmt.Errorf("failed to get gets for folder items: %w", err)
	}

	operationChangeSets := map[UpdateOperation]ChangeSet{}
	for _, item := range items {
		if existing, ok := operationChangeSets[item.Operation]; ok {
			existing.UIDs = append(existing.UIDs, item.ChangeSet.UIDs...)
			operationChangeSets[item.Operation] = existing
		} else {
			operationChangeSets[item.Operation] = item.ChangeSet
		}
	}

	return operationChangeSets, nil
}
