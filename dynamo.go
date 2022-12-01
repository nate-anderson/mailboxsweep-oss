package mailboxsweep

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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

func mergeAttributeValueMaps(maps ...map[string]types.AttributeValue) map[string]types.AttributeValue {
	res := map[string]types.AttributeValue{}
	for _, m := range maps {
		for key, value := range m {
			res[key] = value
		}
	}
	return res
}
