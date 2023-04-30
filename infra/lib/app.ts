import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import { InfraStackProps } from './infra-stack';
import * as route53 from 'aws-cdk-lib/aws-route53';
import { ContainerImage } from 'aws-cdk-lib/aws-ecs';
import { Vpc } from "aws-cdk-lib/aws-ec2";
import * as ECS from "aws-cdk-lib/aws-ecs";
import * as EC2 from 'aws-cdk-lib/aws-ec2';
import * as ECR from 'aws-cdk-lib/aws-ecr';
import { TableStack } from './table';
import { ClassificationQueueStack } from './queues';
import * as S3 from "aws-cdk-lib/aws-s3";
import * as CertMan from "aws-cdk-lib/aws-certificatemanager";
import * as Route53 from "aws-cdk-lib/aws-route53";
import * as Route53Targets from "aws-cdk-lib/aws-route53-targets";
import * as ECSPatterns from "aws-cdk-lib/aws-ecs-patterns";
import * as LoadBalancingV2 from "aws-cdk-lib/aws-elasticloadbalancingv2";
import * as AWSLogs from "aws-cdk-lib/aws-logs";
import * as IAM from "aws-cdk-lib/aws-iam";
import Resources from "../lib/resources";

type AppProps = InfraStackProps & {
    subdomain?: string;
    clusterInstanceType: string;
};

const baseUrl = "mailboxsweep.com";
const latestTag = 'latest';

const Port = {
    HTTPS: 443,
    HTTP: 80
}

export class AppStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props: AppProps) {
        super(scope, id, props);

        const tableStack = new TableStack(this, `app-table-${props.environmentName}`, props);
        const queues = new ClassificationQueueStack(this, `classify-queues-${props.environmentName}`, props);

        const serverRepoName = `mailboxsweep-server`;
        const serverRepo = ECR.Repository.fromRepositoryName(this, serverRepoName, serverRepoName);

        const workerRepoName = `mailboxsweep-worker`;
        const workerRepo = ECR.Repository.fromRepositoryName(this, workerRepoName, workerRepoName);

        const hostedZone = route53.HostedZone.fromHostedZoneAttributes(this, `hosted-zone`, {
            hostedZoneId: Resources.global.dns.zoneId,
            zoneName: Resources.global.dns.fullyQualifiedDomainName,
        });
        const domain: string = Resources[props.environmentName].api.dns.fullyQualifiedDomainName;
        

        const bucket = new S3.Bucket(this, `classifications-${props.environmentName}`, {
            bucketName: `com.mailboxsweep.${props.environmentName}.classification`,
            encryption: S3.BucketEncryption.S3_MANAGED,
            removalPolicy: cdk.RemovalPolicy.RETAIN,
        });

        const vpc = new Vpc(this, `vpc-${props.environmentName}`, {});

        const clusterId = `cluster-${props.environmentName}`;
        const cluster = new ECS.Cluster(this, `mailboxsweep-cluster`, {
            clusterName: clusterId,
            vpc,
            capacity: {
              instanceType: new EC2.InstanceType(props.clusterInstanceType),
            },
            containerInsights: true,
        });

        const serverTaskId = `server-task-${props.environmentName}`
        const serverTaskDefinition = new ECS.TaskDefinition(this, serverTaskId, {
            compatibility: ECS.Compatibility.EC2,
        });
        

        const readSSMPolicy = new IAM.Policy(this, `read-ssm`, {
            policyName: "read-ssm",
            statements: [
                new IAM.PolicyStatement({
                    effect: IAM.Effect.ALLOW,
                    actions: [
                        "ssm:DescribeParameters",
                        "ssm:GetParameters",
                        "ssm:GetParameter",
                        "ssm:GetParametersByPath"
                    ],
                    resources: ["*"],
                })
            ]
        });

        // write to main queue
        const serverWriteQueuePolicy = new IAM.Policy(this, `server-write-queue`, {
            policyName: "write-sqs",
            statements: [
                new IAM.PolicyStatement({
                    effect: IAM.Effect.ALLOW,
                    actions: [
                        "sqs:SendMessage",
                        "sqs:ListQueues",
                        "sqs:GetQueueUrl",
                        "sqs:GetQueueAttributes",
                        "sqs:ListDeadLetterSourceQueues",
                    ],
                    resources: [queues.queue.queueArn],
                })
            ]
        })

        serverTaskDefinition.taskRole.attachInlinePolicy(readSSMPolicy);
        serverTaskDefinition.taskRole.attachInlinePolicy(serverWriteQueuePolicy);

        const appConfig: Record<string, string> = {
            // Generic app config
            ENV: props.environmentName,
            HTTP_PORT: Port.HTTP.toString(),
            DEV_MODE: "false",
            CHANGESET_PAGE_ID_HASH_MOD: "11",
            INBOX_METADATA_TTL_MINUTES: "30",
            CLASSIFICATION_PAGE_SIZE: "30",
            CORS_ALLOWED_ORIGIN: `https://${domain}`,
            MINUTES_BETWEEN_OPEN_REQUESTS: "120",
            MINUTES_BETWEEN_CLOSED_REQUESTS: "60",

            // Login
            BASE_URL: `https://${domain}/login-token`,

            // DB
            DYNAMODB_REGION: "us-west-2",
            DYNAMODB_TABLE: tableStack.table.tableName,

            // Bucket
            S3_REGION: "us-west-2",
            S3_BUCKET: bucket.bucketName,

            // Queues
            SQS_QUEUE_URL: queues.queue.queueUrl,
            SQS_QUEUE: queues.queue.queueName,
            SQS_DLQ_URL: queues.dlq.queueUrl,
            SQS_DLQ: queues.dlq.queueName,
            SQS_REGION: "us-west-2",

            // Sessions & cookies
            COOKIE_DOMAIN: domain,
            SESSION_KEY_LENGTH: "30",
            COOKIE_PATH: "/",
            LOGIN_LINK_DURATION_MINUTES: "10",
            SESSION_DURATION_MINUTES: "180",

            // Stripe
            PAYMENT_ACCESS_DURATION_DAYS: "30",
            STRIPE_PAYMENT_CONFIRMATION_URL: "https://mailboxsweep.com/plan/payment-success?checkoutSessionId={CHECKOUT_SESSION_ID}",
            STRIPE_CHECKOUT_SESSION_DURATION_MINUTES: "30",

            GOOGLE_OAUTH_REDIRECT_URL: `https://${domain}/login-gmail`

            // Secrets that need to come from SSM
            // GOOGLE_OAUTH_CLIENT_ID
            // GOOGLE_OAUTH_CLIENT_SECRET
            // STRIPE_API_KEY
            // CREDENTIAL_CIPHER_KEY_BASE64
            // SIGNING_KEY
        };

        const serverContainer = serverTaskDefinition.addContainer('ServerContainer', {
            image: ContainerImage.fromEcrRepository(serverRepo, latestTag),
            containerName: 'mailboxsweep-server',
            environment: appConfig,
            memoryReservationMiB: 1024,
            logging: ECS.LogDrivers.awsLogs({
                streamPrefix: "mailboxsweep-server",
                logRetention: AWSLogs.RetentionDays.ONE_YEAR,
            }),
            portMappings: [
                {
                    containerPort: Port.HTTP,
                }
            ],
            healthCheck: {
                command: [ `curl localhost/healthcheck` ],
                interval: cdk.Duration.seconds(300),  
            },
        });

        // const serverServiceId = `server-service-${props.environmentName}`;
        // const serverService = new ECS.Ec2Service(this, serverServiceId, {
        //     cluster,
        //     taskDefinition: serverTaskDefinition,
        //     desiredCount: 2,
        // })

        const workerTaskId = `worker-task-${props.environmentName}`
        const workerTaskDefinition = new ECS.TaskDefinition(this, workerTaskId, {
            compatibility: ECS.Compatibility.EC2,
        });

        workerTaskDefinition.addContainer('WorkerContainer', {
            image: ContainerImage.fromEcrRepository(workerRepo, latestTag),
            containerName: 'mailboxsweep-worker',
            environment: {
                ...appConfig,
            },
            memoryReservationMiB: 1024,
            logging: ECS.LogDrivers.awsLogs({
                streamPrefix: "mailboxsweep-worker",
                logRetention: AWSLogs.RetentionDays.ONE_YEAR,
            })
        });
        workerTaskDefinition.taskRole.attachInlinePolicy(readSSMPolicy);

        const workerServiceId = `worker-service-${props.environmentName}`;
        const workerService = new ECS.Ec2Service(this, workerServiceId, {
            cluster,
            taskDefinition: workerTaskDefinition,
            desiredCount: 1,
        });
        
        // read from main queue or DLQ
        const readQueuePolicy = new IAM.Policy(this, `worker-sqs-read`, {
            policyName: "read-queue",
            statements: [
                new IAM.PolicyStatement({
                    effect: IAM.Effect.ALLOW,
                    actions: [
                        "sqs:ListQueues",
                        "sqs:ReceiveMessage",
                        "sqs:GetQueueUrl",
                        "sqs:GetQueueAttributes",
                        "sqs:ListDeadLetterSourceQueues",
                        "sqs:DeleteMessage"
                    ],
                    resources: [queues.queue.queueArn, queues.dlq.queueArn],
                })
            ]
        });

        // write to DLQ
        const writeDLQPolicy = new IAM.Policy(this, `worker-dlq-write`, {
            policyName: "write-dlq",
            statements: [
                new IAM.PolicyStatement({
                    effect: IAM.Effect.ALLOW,
                    actions: [
                        "sqs:SendMessage",
                        "sqs:ListQueues",
                        "sqs:GetQueueUrl",
                        "sqs:GetQueueAttributes",
                        "sqs:ListDeadLetterSourceQueues",
                    ],
                    resources: [queues.dlq.queueArn],
                })
            ]
        });
        workerTaskDefinition.taskRole.attachInlinePolicy(readQueuePolicy);
        workerTaskDefinition.taskRole.attachInlinePolicy(writeDLQPolicy);
 
        const cert = CertMan.Certificate.fromCertificateArn(this, `api-cert-${props.environmentName}`, Resources.global.api.cert.arn);

        const securityGroup = new EC2.SecurityGroup(this, `api-security-group`, {
            vpc,
        });
        securityGroup.addIngressRule(EC2.Peer.ipv4('0.0.0.0/0'), EC2.Port.tcp(Port.HTTP));

        // const serverService = new ECS.Ec2Service(this, `server-service`, {
        //     taskDefinition: serverTaskDefinition,
        //     cluster,
        //     desiredCount: 1,
        //     assignPublicIp: false,
        //     securityGroups: [securityGroup],
        // });

        // serverService.autoScaleTaskCount({ maxCapacity: 3, minCapacity: 1})
        //     .scaleOnCpuUtilization('CpuScaling', {
        //         targetUtilizationPercent: 80,
        //         scaleInCooldown: cdk.Duration.seconds(60),
        //         scaleOutCooldown: cdk.Duration.seconds(60),
        //     });

        // const elb = new LoadBalancingV2.NetworkLoadBalancer(this, `server-lb`, {
        //     vpc,
        //     internetFacing: true,
        // });
        // const listener = elb.addListener(`api`, {
        //     protocol: LoadBalancingV2.Protocol.TLS,
        //     certificates: [cert],
        //     port: Port.HTTPS,
        //     defaultTargetGroups: [new LoadBalancingV2.NetworkTargetGroup(this, `api-target`, {
        //         vpc,
        //         port: Port.HTTP,
        //         preserveClientIp: true,
        //         protocol: LoadBalancingV2.Protocol.TCP,
        //         targets: [serverService],
        //         healthCheck: {
        //             path: "/healthcheck",
        //             port: Port.HTTP.toString(),
        //             protocol: LoadBalancingV2.Protocol.HTTP,
        //         }
        //     })]
        // });

        const loadBalancedAPIService = new ECSPatterns.ApplicationLoadBalancedEc2Service(this, `server-service`, {
            cluster,
            taskDefinition: serverTaskDefinition,
            cpu: 256,
            memoryReservationMiB: 512,
            desiredCount: 1,
            protocol: LoadBalancingV2.ApplicationProtocol.HTTPS,
            openListener: true,
            domainZone: hostedZone,
            certificate: cert,
            publicLoadBalancer: true,
            maxHealthyPercent: 200,
            minHealthyPercent: 100,
        });
        loadBalancedAPIService.targetGroup.configureHealthCheck({
            path: "/healthcheck",
            port: 'traffic-port',
            protocol: LoadBalancingV2.Protocol.HTTP,
            interval: cdk.Duration.seconds(300),
            healthyThresholdCount: 2,
            unhealthyThresholdCount: 2,
        });
        


        new Route53.ARecord(this, `api-a-record`, {
            zone: hostedZone,
            recordName: domain,
            target: route53.RecordTarget.fromAlias(new Route53Targets.LoadBalancerTarget(loadBalancedAPIService.loadBalancer)),
        })
    }
}