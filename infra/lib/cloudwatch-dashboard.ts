import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import { InfraStackProps } from './infra-stack';
import * as SQS from "aws-cdk-lib/aws-sqs";
import * as DynamoDB from "aws-cdk-lib/aws-dynamodb";
import * as CloudWatch from 'aws-cdk-lib/aws-cloudwatch';

type CloudwatchDashboardProps = InfraStackProps & {
    table: DynamoDB.Table,
    queue: SQS.Queue,
    dlq: SQS.Queue,
}

export class CloudwatchDashboard extends cdk.Stack {
    constructor(scope: Construct, id: string, props: CloudwatchDashboardProps) {
        super(scope, id, props);

        const dashboard = new CloudWatch.Dashboard(this, `app-dash`, {
            dashboardName: `MailboxSweep_${props.environmentName}`,
        });

        const dbCapacityGraph = new CloudWatch.GraphWidget({
            left: [
                new CloudWatch.Metric({
                    metricName: "ConsumedWriteCapacityUnits",
                    namespace: "AWS/DynamoDB",
                    dimensionsMap: {
                        TableName: props.table.tableName,
                    },
                    unit: CloudWatch.Unit.COUNT,
                    label: "DynamoDB Write Capacity",
                }),
                new CloudWatch.Metric({
                    metricName: "ConsumedReadCapacityUnits",
                    namespace: "AWS/DynamoDB",
                    dimensionsMap: {
                        TableName: props.table.tableName,
                    },
                    unit: CloudWatch.Unit.COUNT,
                    label: "DynamoDB Read Capacity",
                }),
            ],
        });

        const dbErrorsGraph = new CloudWatch.GraphWidget({
            left: [
                new CloudWatch.Metric({
                    metricName: "SystemErrors",
                    namespace: "AWS/DynamoDB",
                    dimensionsMap: {
                        TableName: props.table.tableName,
                    },
                    unit: CloudWatch.Unit.COUNT,
                    statistic: "avg",
                }),
                new CloudWatch.Metric({
                    metricName: "UserErrors",
                    namespace: "AWS/DynamoDB",
                    dimensionsMap: {
                        TableName: props.table.tableName,
                    },
                    unit: CloudWatch.Unit.COUNT,
                    statistic: "avg",
                })
            ]
        })

        const queueBacklogGraph = new CloudWatch.GraphWidget({
            left: [
                new CloudWatch.Metric({
                    metricName: "ApproximateNumberOfMessagesVisible",
                    namespace: "AWS/SQS"
                })
            ]
        })

        dashboard.addWidgets(
            dbCapacityGraph,
            dbErrorsGraph,
        );
    }
}