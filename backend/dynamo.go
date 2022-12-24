package mailboxsweep

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	DynamoPK      string = "pk"
	DynamoSK      string = "sk"
	DynamoGSI1    string = "GSI1"
	DynamoGSI1PK  string = "gsi1pk"
	DynamoGSI1SK  string = "gsi1sk"
	DynamoLSI1SK  string = "lsi1sk"
	DynamoExpires string = "expires_at"
	DynamoLSI1    string = "LSI1"
)

var (
	DynamoTimeFormat  = time.RFC3339
	DynamoUserDataTTL = time.Hour * 24 * 30 // 30 days
)

func MergeAttributeValueMaps(maps ...map[string]types.AttributeValue) map[string]types.AttributeValue {
	res := map[string]types.AttributeValue{}
	for _, m := range maps {
		for key, value := range m {
			res[key] = value
		}
	}
	return res
}

func CreateDynamoDBTableDefinition(table string) *dynamodb.CreateTableInput {
	var tableDef = &dynamodb.CreateTableInput{
		TableName: &table,
		ProvisionedThroughput: &dynamoTypes.ProvisionedThroughput{
			ReadCapacityUnits:  ptr[int64](10),
			WriteCapacityUnits: ptr[int64](10),
		},
		AttributeDefinitions: []dynamoTypes.AttributeDefinition{
			{
				AttributeName: &DynamoPK,
				AttributeType: dynamoTypes.ScalarAttributeTypeS,
			},
			{
				AttributeName: &DynamoSK,
				AttributeType: dynamoTypes.ScalarAttributeTypeS,
			},
			{
				AttributeName: &DynamoGSI1PK,
				AttributeType: dynamoTypes.ScalarAttributeTypeS,
			},
			{
				AttributeName: &DynamoGSI1SK,
				AttributeType: dynamoTypes.ScalarAttributeTypeS,
			},
			{
				AttributeName: &DynamoLSI1SK,
				AttributeType: dynamoTypes.ScalarAttributeTypeS,
			},
		},
		KeySchema: []dynamoTypes.KeySchemaElement{
			{
				AttributeName: &DynamoPK,
				KeyType:       dynamoTypes.KeyTypeHash,
			},
			{
				AttributeName: &DynamoSK,
				KeyType:       dynamoTypes.KeyTypeRange,
			},
		},
		GlobalSecondaryIndexes: []dynamoTypes.GlobalSecondaryIndex{
			{
				IndexName: &DynamoGSI1,
				KeySchema: []dynamoTypes.KeySchemaElement{
					{
						AttributeName: &DynamoGSI1PK,
						KeyType:       dynamoTypes.KeyTypeHash,
					},
					{
						AttributeName: &DynamoGSI1SK,
						KeyType:       dynamoTypes.KeyTypeRange,
					},
				},
				Projection: &dynamoTypes.Projection{
					ProjectionType: dynamoTypes.ProjectionTypeKeysOnly,
				},
				ProvisionedThroughput: &dynamoTypes.ProvisionedThroughput{
					ReadCapacityUnits:  ptr[int64](10),
					WriteCapacityUnits: ptr[int64](10),
				},
			},
		},
		LocalSecondaryIndexes: []dynamoTypes.LocalSecondaryIndex{
			{
				IndexName: &DynamoLSI1,
				KeySchema: []dynamoTypes.KeySchemaElement{
					{
						AttributeName: &DynamoPK,
						KeyType:       dynamoTypes.KeyTypeHash,
					},
					{
						AttributeName: &DynamoLSI1SK,
						KeyType:       dynamoTypes.KeyTypeRange,
					},
				},
				Projection: &dynamoTypes.Projection{
					ProjectionType: dynamoTypes.ProjectionTypeKeysOnly,
				},
			},
		},
	}
	return tableDef
}
