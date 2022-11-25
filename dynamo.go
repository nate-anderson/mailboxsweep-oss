package mailboxsweep

import "time"

var (
	DynamoPK      string = "pk"
	DynamoSK      string = "sk"
	DynamoGSI1PK  string = "gsi1pk"
	DynamoGSI1SK  string = "gsi1sk"
	DynamoExpires string = "expires_at"
)

var (
	dynamoTimeFormat  = time.RFC3339
	dynamoUserDataTTL = time.Hour * 24 * 30 // 30 days
)
