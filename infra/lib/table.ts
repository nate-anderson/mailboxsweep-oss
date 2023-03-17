import * as cdk from 'aws-cdk-lib';
import * as DynamoDB from 'aws-cdk-lib/aws-dynamodb';
import { Construct } from 'constructs';
import { InfraStackProps } from './infra-stack';

export class TableStack extends cdk.Stack {
    public readonly tableName: string;
    public readonly tableArn: string;

    constructor(scope: Construct, id: string, props: InfraStackProps) {
        super(scope, id, props);

        this.tableName = `mailboxsweep_table_${props.environmentName}`;
        const table = new DynamoDB.Table(this, this.tableName, {
            tableName: this.tableName,
            billingMode: DynamoDB.BillingMode.PROVISIONED,
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
        this.tableArn = table.tableArn;

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