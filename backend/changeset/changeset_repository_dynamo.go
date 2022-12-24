package changeset

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"mailboxsweep"
	"strconv"
	"strings"
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

	lsi1 string `dynamodbav:"lsi1sk"`
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

func (d DynamoChangesetRepository) ListByInbox(ctx context.Context, inbox string, consistentRead bool) ([]ChangeSet, error) {
	res, err := d.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &d.tableName,
		KeyConditionExpression: aws.String("#pk = :email"),
		ExpressionAttributeNames: map[string]string{
			"#pk": mailboxsweep.DynamoPK,
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
			existing.Paths = append(existing.Paths, c.Paths...)
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

func (d DynamoChangesetRepository) partitionID(path string) (uint64, error) {
	hexVal := md5.Sum([]byte(path))
	hexStr := hex.EncodeToString(hexVal[:4])
	dec, err := strconv.ParseUint(hexStr, 16, 64)
	if err != nil {
		return 0, err
	}
	return dec % d.pageIDHashMod, nil
}

func (d DynamoChangesetRepository) partitionSet(changeSet ChangeSet) (map[string]ChangeSet, error) {
	partitionMap := make(map[uint64][]string)
	for _, path := range changeSet.Paths {
		partitionID, err := d.partitionID(path)
		if err != nil {
			return nil, err
		}
		partitionMap[partitionID] = append(partitionMap[partitionID], path)
	}

	res := make(map[string]ChangeSet)
	for partitionID, paths := range partitionMap {
		sk := d.formatSK(changeSet.Folder, changeSet.Operation, partitionID)
		c := changeSet
		c.Paths = paths
		res[sk] = c
	}
	return res, nil
}

// Append to a changeset, splitting it into pages using the hash function
func (d DynamoChangesetRepository) Append(ctx context.Context, changeSet ChangeSet) (int, error) {
	now := time.Now()
	expires := now.Add(mailboxsweep.DynamoUserDataTTL).Unix()

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
				#folder = :folder,
				#operation = :operation,
				#created = if_not_exists(created, :created),
				#updated = :updated,
				#expires = :expires
				
				ADD #paths :paths`,
			),
			ExpressionAttributeNames: map[string]string{
				"#inbox":     "inbox",
				"#lsi1":      mailboxsweep.DynamoLSI1SK,
				"#folder":    "folder",
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
				":folder": &types.AttributeValueMemberS{
					Value: changeSet.Folder,
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
					Value: d.formatLSI1(changeSet.Folder),
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

func (d DynamoChangesetRepository) Remove(ctx context.Context, changeSet ChangeSet) (int, error) {
	now := time.Now()
	expires := now.Add(mailboxsweep.DynamoUserDataTTL).Unix()

	// track diff
	oldPaths := make([]string, 0, len(changeSet.Paths))

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
			diff++
		}
	}
	return diff, nil
}

func (d DynamoChangesetRepository) FindByInboxFolder(ctx context.Context, inbox, folder string, consistentRead bool) (map[UpdateOperation]ChangeSet, error) {
	keysRes, err := d.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &d.tableName,
		ConsistentRead:         &consistentRead,
		KeyConditionExpression: aws.String("#pk = :pk AND #lsi1 = :lsi1"),
		ExpressionAttributeNames: map[string]string{
			"#pk":   mailboxsweep.DynamoPK,
			"#lsi1": mailboxsweep.DynamoLSI1SK,
		},
		IndexName: &mailboxsweep.DynamoLSI1,
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
			mailboxsweep.DynamoPK: &types.AttributeValueMemberS{
				Value: item.PK,
			},
			mailboxsweep.DynamoSK: &types.AttributeValueMemberS{
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
			existing.Paths = append(existing.Paths, item.ChangeSet.Paths...)
			operationChangeSets[item.Operation] = existing
		} else {
			operationChangeSets[item.Operation] = item.ChangeSet
		}
	}

	return operationChangeSets, nil
}
