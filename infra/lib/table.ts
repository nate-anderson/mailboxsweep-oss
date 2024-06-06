import * as cdk from 'aws-cdk-lib';
import * as DynamoDB from 'aws-cdk-lib/aws-dynamodb';
import { Construct } from 'constructs';

export type TableStackProps = cdk.StackProps & {
    environmentName: string;
}

export class TableStack extends cdk.Stack {
    public readonly table: DynamoDB.Table;

    constructor(scope: Construct, id: string, props: TableStackProps) {
        super(scope, id, props);

        const tableName = `mailboxsweep_table_${props.environmentName}`;
        const table = new DynamoDB.Table(this, tableName, {
            tableName: tableName,
            billingMode: DynamoDB.BillingMode.PROVISIONED,
            writeCapacity: 10,
            readCapacity: 10,
            pointInTimeRecovery: true,
            partitionKey: {
                name: "pk",
                type: DynamoDB.AttributeType.STRING,
            },
            sortKey: {
                name: "sk",
                type: DynamoDB.AttributeType.STRING,
            },
            removalPolicy: cdk.RemovalPolicy.RETAIN,
            timeToLiveAttribute: "expires_at",
        });
        this.table = table;

        table.addGlobalSecondaryIndex({
            indexName: "GSI1",
            partitionKey: {
                type: DynamoDB.AttributeType.STRING,
                name: "gsi1pk"
            },
            sortKey: {
                type: DynamoDB.AttributeType.STRING,
                name: "gsi1sk",
            },
            readCapacity: 5,
            writeCapacity: 3,
            projectionType: DynamoDB.ProjectionType.KEYS_ONLY,
        });

        table.addLocalSecondaryIndex({
            indexName: "LSI1",
            sortKey: {
                name: "lsi1sk",
                type: DynamoDB.AttributeType.STRING
            },
        });
    }
}