import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import { InfraStackProps } from './infra-stack';
import * as SQS from "aws-cdk-lib/aws-sqs";
import * as DynamoDB from "aws-cdk-lib/aws-dynamodb";
import * as CloudWatch from 'aws-cdk-lib/aws-cloudwatch';
import * as AWSLogs from 'aws-cdk-lib/aws-logs';
import * as ElasticLoadBalancingV2 from 'aws-cdk-lib/aws-elasticloadbalancingv2';

type CloudwatchDashboardProps = InfraStackProps & {
    table: DynamoDB.Table,
    queue: SQS.Queue,
    dlq: SQS.Queue,
    serverLogGroup: AWSLogs.ILogGroup,
    workerLogGroup: AWSLogs.ILogGroup,
    loadBalancer: ElasticLoadBalancingV2.ApplicationLoadBalancer,
}

export class CloudwatchDashboard extends cdk.Stack {
    public readonly dashboard: CloudWatch.Dashboard;
    constructor(scope: Construct, id: string, props: CloudwatchDashboardProps) {
        super(scope, id, props);

        const dashboard = new CloudWatch.Dashboard(this, `app-dash`, {
            dashboardName: `MailboxSweep_${props.environmentName}`,
            start: "-P1W"
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
        });

        const serverErrorMetricFilter = new AWSLogs.MetricFilter(this, `server-error-logs`, {
            logGroup: props.serverLogGroup,
            filterPattern: {
                logPatternString: `{ ($.level = "error") }`,
            },
            metricNamespace: `${props.environmentName}-server-errors`,
            metricName: "error-level-logs-server",
            metricValue: "1",
            defaultValue: 0,
        });

        const serverErrorsGraph = new CloudWatch.GraphWidget({
            left: [
                serverErrorMetricFilter.metric({
                    period: cdk.Duration.minutes(1),
                    statistic: "max",
                    unit: CloudWatch.Unit.COUNT,
                }),
            ]
        });

        const workerErrorMetricFilter = new AWSLogs.MetricFilter(this, `worker-error-logs`, {
            logGroup: props.workerLogGroup,
            filterPattern: {
                logPatternString: `{ ($.level = "error") }`,
            },
            metricNamespace: `${props.environmentName}-worker-errors`,
            metricName: "error-level-logs-worker",
            metricValue: "1",
            defaultValue: 0,
        });

        const workerErrorsGraph = new CloudWatch.GraphWidget({
            left: [
                workerErrorMetricFilter.metric({
                    period: cdk.Duration.minutes(1),
                    statistic: "max",
                    unit: CloudWatch.Unit.COUNT,
                })
            ]
        })

        // const loadbalancerErrorsMetric = new CloudWatch.Metric({
        //     namespace: 'AWS/ApplicationELB',
        //     metricName: "HTTPCode_ELB_4XX_Count",
        //     dimensionsMap: {
        //         LoadBalancer: props.loadBalancer.loadBalancerArn,
        //     }
        // });

        dashboard.addWidgets(
            dbCapacityGraph,
            dbErrorsGraph,
            queueBacklogGraph,
            serverErrorsGraph,
            workerErrorsGraph,
        );

        this.dashboard = dashboard;
    }
}