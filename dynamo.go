package mailboxsweep

import "time"

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
	dynamoTimeFormat  = time.RFC3339
	dynamoUserDataTTL = time.Hour * 24 * 30 // 30 days
)
