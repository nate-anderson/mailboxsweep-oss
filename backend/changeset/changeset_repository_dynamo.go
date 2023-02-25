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

func (d DynamoChangesetRepository) FindByInboxAndOperation(ctx context.Context, inbox string, operation UpdateOperation, consistentRead bool) (*ChangeSet, error) {
	res, err := d.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &d.tableName,
		KeyConditionExpression: aws.String("#pk = :email"),
		ExpressionAttributeNames: map[string]string{
			"#pk": mailboxsweep.DynamoPK,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":email": &types.AttributeValueMemberS{
				Value: d.formatPK(inbox, operation),
			},
		},
		ConsistentRead: &consistentRead,
	})
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	results := make([]changeSetDocument, len(res.Items))
	err = attributevalue.UnmarshalListOfMaps(res.Items, &results)
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

func (d DynamoChangesetRepository) formatPK(inbox string, operation UpdateOperation) string {
	return fmt.Sprintf("CHANGESET#%s#%s", inbox, operation)
}

func (d DynamoChangesetRepository) formatSK(partitionID uint64) string {
	return fmt.Sprintf("PAGE#%d", partitionID)
}

func (d DynamoChangesetRepository) formatLSI1(operation UpdateOperation) string {
	return fmt.Sprintf("OPERATION#%s", operation)
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
		sk := d.formatSK(partitionID)
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

	pk := d.formatPK(changeSet.Inbox, changeSet.Operation)

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
					Value: d.formatLSI1(changeSet.Operation),
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

	pk := d.formatPK(changeSet.Inbox, changeSet.Operation)

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
			diff--
		}
	}
	return diff, nil
}
