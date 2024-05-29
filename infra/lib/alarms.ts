import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import { InfraStackProps } from './infra-stack';
import * as SQS from 'aws-cdk-lib/aws-sqs';
import * as CloudWatch from "aws-cdk-lib/aws-cloudwatch";
import * as Actions from "aws-cdk-lib/aws-cloudwatch-actions";
import * as SNS from "aws-cdk-lib/aws-sns";
import * as Subscriptions from "aws-cdk-lib/aws-sns-subscriptions";
import * as LBv2 from "aws-cdk-lib/aws-elasticloadbalancingv2";

type AlarmsStackProps = InfraStackProps & {
    classifyQueue: SQS.IQueue;
    commitQueue: SQS.IQueue;
    classifyDLQ: SQS.IQueue;
    commitDLQ: SQS.IQueue;
    emailAddr: string;
    loadBalancer: LBv2.ApplicationLoadBalancer;
};

export class AlarmsStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props: AlarmsStackProps) {
        super(scope, id, props);

        const maxClassifyQueueMessageAge = 60 * 60 * 3; // 3 hours
        const classifyQueueAlarms = new QueueMetricAlarms(this, `${props.environmentName}-classify`, {
            queue: props.classifyQueue,
            dlq: props.classifyDLQ,
            maxDLQMessageAgeSeconds: maxClassifyQueueMessageAge * 3,
            maxQueueMessageAgeSeconds: maxClassifyQueueMessageAge,
            ...props,
        });

        const maxCommitQueueMessageAge = 60 * 30; // 30 minutes
        const commitQueueAlarms = new QueueMetricAlarms(this, `${props.environmentName}-commit`, {
            queue: props.commitQueue,
            dlq: props.commitDLQ,
            maxDLQMessageAgeSeconds: maxCommitQueueMessageAge * 3,
            maxQueueMessageAgeSeconds: maxCommitQueueMessageAge,
            ...props,
        });

        new LoadBalancerAlarms(this, `${props.environmentName}-alarms`, {
            ...props,
            loadBalancer: props.loadBalancer,
        })
    }
}

type QueueMetricAlarmsStackProps = InfraStackProps & {
    queue: SQS.IQueue;
    dlq: SQS.IQueue | undefined;
    maxQueueMessageAgeSeconds: number;
    maxDLQMessageAgeSeconds: number | undefined;
    emailAddr: string;
}

class QueueMetricAlarms extends cdk.Stack {
    constructor(scope: Construct, description: string, props: QueueMetricAlarmsStackProps) {
        super(scope, `${description}-alarm-stack`, props);

        const oldestMessageMetric = new CloudWatch.Metric({
            metricName: "ApproximateAgeOfOldestMessage",
            namespace: "AWS/SQS",
            statistic: "max",
            dimensionsMap: {
                QueueName: props.queue.queueName,
            }
        });

        const oldestMessageAlarm = new CloudWatch.Alarm(this, `oldest-${description}-queue-msg`, {
            alarmName: `Oldest ${description} Q MSG`,
            alarmDescription: `Oldest message in ${description} queue`,
            metric: oldestMessageMetric,
            threshold: props.maxQueueMessageAgeSeconds,
            evaluationPeriods: 1,
            actionsEnabled: true,
        });

        const topic = new SNS.Topic(scope, `${description}-alarm-topic-${props.environmentName}`, {
            displayName: `${description} Alarm Topic`,
        });
        topic.addSubscription(new Subscriptions.EmailSubscription(props.emailAddr, {
            json: true,
        }))

        oldestMessageAlarm.addAlarmAction(new Actions.SnsAction(topic))

        if (props.dlq) {
            const oldestDlqMessageMetric = new CloudWatch.Metric({
                metricName: "ApproximateAgeOfOldestMessage",
                namespace: "AWS/SQS",
                statistic: "max",
                dimensionsMap: {
                    QueueName: props.dlq.queueName,
                }
            });

            const oldestDlqMessageAlarm = new CloudWatch.Alarm(this, `oldest-${description}-dlq-queue-msg`, {
                alarmName: `Oldest ${description} DLQ MSG`,
                alarmDescription: `Oldest message in ${description} DLQ`,
                metric: oldestDlqMessageMetric,
                threshold: props.maxDLQMessageAgeSeconds!,
                evaluationPeriods: 1,
                actionsEnabled: true,
            })

            const dlqTopic = new SNS.Topic(scope, `${description}-dlq-alarm-topic-${props.environmentName}`, {
                displayName: `${description} DLQ Alarm Topic`,
            });
            dlqTopic.addSubscription(new Subscriptions.EmailSubscription(props.emailAddr, {
                json: true,
            }));
            oldestDlqMessageAlarm.addAlarmAction(new Actions.SnsAction(dlqTopic));
        }
    }
}

type LoadBalancerAlarmsStackProps = InfraStackProps & {
    loadBalancer: LBv2.ApplicationLoadBalancer;
    emailAddr: string;
}

class LoadBalancerAlarms extends cdk.Stack {
    constructor(scope: Construct, description: string, props: LoadBalancerAlarmsStackProps) {
        super(scope, `${description}-lb-alarm-stack`, props);

        const loadbalancer4xxErrorsMetric = new CloudWatch.Metric({
            namespace: 'AWS/ApplicationELB',
            metricName: "HTTPCode_ELB_4XX_Count",
            dimensionsMap: {
                LoadBalancer: props.loadBalancer.loadBalancerArn,
            }
        });
        const loadbalancer4xxErrorsAlarm = new CloudWatch.Alarm(this, `${description}-lb-4xx-alarm`, {
            alarmName: "ALB 4xx Count",
            alarmDescription: "ALB 4xx errors",
            metric: loadbalancer4xxErrorsMetric,
            threshold: 3,
            evaluationPeriods: 1,
            treatMissingData: CloudWatch.TreatMissingData.IGNORE,
            actionsEnabled: true,
        });
        const alb4xxTopic = new SNS.Topic(this, `${description}-alb-4xx-topic`, {
            displayName: `ALB 4xx Count`,
        });
        alb4xxTopic.addSubscription(new Subscriptions.EmailSubscription(props.emailAddr, { json: true }));
        loadbalancer4xxErrorsAlarm.addAlarmAction(new Actions.SnsAction(alb4xxTopic));

        const httpService4xxErrorsMetric = new CloudWatch.Metric({
            namespace: 'AWS/ApplicationELB',
            metricName: "HTTPCode_Target_4XX_Count",
            dimensionsMap: {
                LoadBalancer: props.loadBalancer.loadBalancerArn,
            }
        });
        const httpService4xxErrorsAlarm = new CloudWatch.Alarm(this, `${description}-http-4xx-alarm`, {
            alarmName: "HTTP Service Target 4xx Count",
            alarmDescription: "HTTP Service 4xx errors caught by LB",
            metric: httpService4xxErrorsMetric,
            threshold: 3,
            evaluationPeriods: 1,
            treatMissingData: CloudWatch.TreatMissingData.IGNORE,
            actionsEnabled: true,
        });
        const http4xxTopic = new SNS.Topic(this, `${description}-http-4xx-topic`, {
            displayName: `HTTP Service 4xx Count`,
        });
        http4xxTopic.addSubscription(new Subscriptions.EmailSubscription(props.emailAddr, { json: true }));
        httpService4xxErrorsAlarm.addAlarmAction(new Actions.SnsAction(http4xxTopic));

        const loadbalancer5xxErrorsMetric = new CloudWatch.Metric({
            namespace: 'AWS/ApplicationELB',
            metricName: "HTTPCode_ELB_5XX_Count",
            dimensionsMap: {
                LoadBalancer: props.loadBalancer.loadBalancerArn,
            }
        });
        const loadbalancer5xxErrorsAlarm = new CloudWatch.Alarm(this, `${description}-lb-5xx-alarm`, {
            alarmName: "ALB 5xx Count",
            alarmDescription: "ALB 5xx errors",
            metric: loadbalancer5xxErrorsMetric,
            threshold: 2,
            evaluationPeriods: 1,
            treatMissingData: CloudWatch.TreatMissingData.IGNORE,
            actionsEnabled: true,
        });
        const alb5xxTopic = new SNS.Topic(this, `${description}-alb-5xx-topic`, {
            displayName: `ALB 5xx Count`,
        });
        alb5xxTopic.addSubscription(new Subscriptions.EmailSubscription(props.emailAddr, { json: true }));
        loadbalancer5xxErrorsAlarm.addAlarmAction(new Actions.SnsAction(alb5xxTopic));


        const httpService5xxErrorsMetric = new CloudWatch.Metric({
            namespace: 'AWS/ApplicationELB',
            metricName: "HTTPCode_Target_5XX_Count",
            dimensionsMap: {
                LoadBalancer: props.loadBalancer.loadBalancerArn,
            }
        });
        const httpService5xxErrorsAlarm = new CloudWatch.Alarm(this, `${description}-http-5xx-alarm`, {
            alarmName: "HTTP Service Target 5xx Count",
            alarmDescription: "HTTP Service 5xx errors caught by LB",
            metric: httpService5xxErrorsMetric,
            threshold: 2,
            evaluationPeriods: 1,
            treatMissingData: CloudWatch.TreatMissingData.IGNORE,
            actionsEnabled: true,
        });
        const http5xxTopic = new SNS.Topic(this, `${description}-http-5xx-topic`, {
            displayName: `HTTP Service 5xx Count`,
        });
        http5xxTopic.addSubscription(new Subscriptions.EmailSubscription(props.emailAddr, { json: true }));
        httpService5xxErrorsAlarm.addAlarmAction(new Actions.SnsAction(http5xxTopic));

        const latencyMetric = new CloudWatch.Metric({
            namespace: "AWS/ApplicationELB",
            metricName: "TargetResponseTime",
            dimensionsMap: {
                LoadBalancer: props.loadBalancer.loadBalancerArn,
            },
            statistic: "avg",
            period: cdk.Duration.minutes(2),
        });
        const latencyTopic = new SNS.Topic(this, `${description}-latency-topic`, {
            displayName: `Server Latency`,
        });
        const latencyAlarm = new CloudWatch.Alarm(this, `${description}-latency-alarm`, {
            alarmName: "Latency",
            alarmDescription: "Latency",
            metric: latencyMetric,
            threshold: 1,
            evaluationPeriods: 1,
            treatMissingData: CloudWatch.TreatMissingData.IGNORE,
            actionsEnabled: true,
        });
        latencyTopic.addSubscription(new Subscriptions.EmailSubscription(props.emailAddr, { json: true }));
        latencyAlarm.addAlarmAction(new Actions.SnsAction(latencyTopic));


    }
}