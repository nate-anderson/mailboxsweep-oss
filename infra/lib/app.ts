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
import * as CFOrigins from 'aws-cdk-lib/aws-cloudfront-origins';
import * as CloudFront from 'aws-cdk-lib/aws-cloudfront';
import * as S3Deployment from 'aws-cdk-lib/aws-s3-deployment';
import { CloudwatchDashboard } from './cloudwatch-dashboard';
import { join } from 'path';

type AppProps = InfraStackProps & {
    subdomain?: string;
    clusterInstanceType: string;
};

const Port = {
    HTTPS: 443,
    HTTP: 80
}

export class AppStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props: AppProps) {
        super(scope, id, props);

        const tableStack = new TableStack(this, `app-table-${props.environmentName}`, props);
        const queues = new ClassificationQueueStack(this, `classify-queues-${props.environmentName}`, props);

        const serverRepoName = `mailboxsweep-server-${props.environmentName}`;
        const serverRepo = ECR.Repository.fromRepositoryName(this, serverRepoName, serverRepoName);

        const workerRepoName = `mailboxsweep-worker-${props.environmentName}`;
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
            lifecycleRules: [
                {
                    id: "ttl",
                    expiration: cdk.Duration.days(30),
                }
            ]
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

        const webUrl = Resources[props.environmentName].web.dns.fullyQualifiedDomainName;

        const appConfig: Record<string, string> = {
            // Generic app config
            ENV: props.environmentName,
            HTTP_PORT: Port.HTTP.toString(),
            STAGING_ALLOWED_USERS: "nathanid95@gmail.com,camerah91@gmail.com,citrusnewsletter@gmail.com",
            DEV_MODE: "false",
            CHANGESET_PAGE_ID_HASH_MOD: "11",
            INBOX_METADATA_TTL_MINUTES: "30",
            CLASSIFICATION_PAGE_SIZE: "30",
            CORS_ALLOWED_ORIGIN: `https://${webUrl}`,
            MINUTES_BETWEEN_OPEN_REQUESTS: "120",
            MINUTES_BETWEEN_CLOSED_REQUESTS: "10",

            // Login
            BASE_URL: `https://${webUrl}/login/token`,

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

            SES_REGION: "us-west-2",
            SES_LOGIN_EMAIL_TEMPLATE_NAME: "mailboxsweep_login_link",

            // Sessions & cookies
            COOKIE_DOMAIN: webUrl,
            SESSION_KEY_LENGTH: "30",
            COOKIE_PATH: "/",
            LOGIN_LINK_DURATION_MINUTES: "10",
            SESSION_DURATION_MINUTES: "180",

            // Stripe
            PAYMENT_ACCESS_DURATION_DAYS: "30",
            STRIPE_PAYMENT_CONFIRMATION_URL: `https://${webUrl}/plan/payment-success?checkoutSessionId={CHECKOUT_SESSION_ID}`,
            STRIPE_CHECKOUT_SESSION_DURATION_MINUTES: "30",

            GOOGLE_OAUTH_REDIRECT_URL: `https://${webUrl}/login/gmail`

            // Secrets that need to come from SSM
            // GOOGLE_OAUTH_CLIENT_ID
            // GOOGLE_OAUTH_CLIENT_SECRET
            // STRIPE_API_KEY
            // CREDENTIAL_CIPHER_KEY_BASE64
            // SIGNING_KEY
        };

        // environment-specific "latest" tag for ECR images
        const latestTag = `latest`;
        const serverLogGroup = new AWSLogs.LogGroup(this, `server-logs`, {
            logGroupName: `mailboxsweep-server-${props.environmentName}`,
            removalPolicy: cdk.RemovalPolicy.RETAIN,
            retention: AWSLogs.RetentionDays.ONE_YEAR,
        });

        const serverContainer = serverTaskDefinition.addContainer('ServerContainer', {
            image: ContainerImage.fromEcrRepository(serverRepo, latestTag),
            containerName: 'mailboxsweep-server',
            environment: {
                ...appConfig,
                FORCE_BUILD_REFRESH: "t",
            },
            memoryReservationMiB: 1024,
            logging: ECS.LogDrivers.awsLogs({
                streamPrefix: "mailboxsweep-server",
                logGroup: serverLogGroup,
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

        // server must be able to read and serve classification trees
        bucket.grantRead(serverTaskDefinition.taskRole);

        // const serverServiceId = `server-service-${props.environmentName}`;
        // const serverService = new ECS.Ec2Service(this, serverServiceId, {
        //     cluster,
        //     taskDefinition: serverTaskDefinition,
        //     desiredCount: 2,
        // })

        // write to DLQ
        const sendEmailPolicy = new IAM.Policy(this, `taskdef-ses-send`, {
            policyName: "send-email",
            statements: [
                new IAM.PolicyStatement({
                    effect: IAM.Effect.ALLOW,
                    actions: [
                        "ses:SendEmail",
                        "ses:SendRawEmail",
                        "ses:SendTemplatedEmail",
                    ],
                    resources: ['*'],
                })
            ]
        });
        serverTaskDefinition.taskRole.attachInlinePolicy(sendEmailPolicy);

        const workerTaskId = `worker-task`;
        const workerTaskDefinition = new ECS.TaskDefinition(this, workerTaskId, {
            compatibility: ECS.Compatibility.EC2,
        });

        const workerLogGroup = new AWSLogs.LogGroup(this, `worker-logs`, {
            logGroupName: `mailboxsweep-worker-${props.environmentName}`,
            removalPolicy: cdk.RemovalPolicy.RETAIN,
            retention: AWSLogs.RetentionDays.ONE_YEAR,
        });

        workerTaskDefinition.addContainer('WorkerContainer', {
            image: ContainerImage.fromEcrRepository(workerRepo, latestTag),
            containerName: 'mailboxsweep-worker',
            environment: {
                ...appConfig,
                FORCE_BUILD_REFRESH: "t",
            },
            memoryReservationMiB: 1024,
            logging: ECS.LogDrivers.awsLogs({
                streamPrefix: "mailboxsweep-worker",
                logGroup: workerLogGroup,
            })
        });
        workerTaskDefinition.taskRole.attachInlinePolicy(readSSMPolicy);

        const workerServiceId = `worker-service`;
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
        workerTaskDefinition.taskRole.attachInlinePolicy(sendEmailPolicy);
 
        const cert = CertMan.Certificate.fromCertificateArn(this, `api-cert-${props.environmentName}`, Resources.global.api.cert.arn);

        const securityGroup = new EC2.SecurityGroup(this, `api-security-group`, {
            vpc,
        });
        securityGroup.addIngressRule(EC2.Peer.ipv4('0.0.0.0/0'), EC2.Port.tcp(Port.HTTP));


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
        
        tableStack.table.grantReadWriteData(serverTaskDefinition.taskRole);
        tableStack.table.grantReadWriteData(workerTaskDefinition.taskRole);

        bucket.grantPut(workerTaskDefinition.taskRole);
        

        new Route53.ARecord(this, `api-a-record`, {
            zone: hostedZone,
            recordName: domain,
            target: route53.RecordTarget.fromAlias(new Route53Targets.LoadBalancerTarget(loadBalancedAPIService.loadBalancer)),
        });

        const cloudFrontCert = new CertMan.DnsValidatedCertificate(this, "web-cert", {
            domainName: webUrl,
            hostedZone: hostedZone,
            region: "us-east-1",  //standard for acm certs
        });

        // frontend
        const staticSiteBucket = S3.Bucket.fromBucketName(this, `static-site-bucket`, `com.mailboxsweep.${props.environmentName}.web`);
        const buildBucket = S3.Bucket.fromBucketName(this, `builds-bucket`, `com.mailboxsweep.builds.${props.environmentName}.web`)
        
        const originAccessIdentity = CloudFront.OriginAccessIdentity.fromOriginAccessIdentityId(this, `site-origin`, Resources[props.environmentName].web.cdn.originAccessId);

        // rewrite request URIs so that `login` becomes `/login/index.html`
        const rewriteURIFunc = new CloudFront.Function(this, `url-rewrite`, {
            code: CloudFront.FunctionCode.fromFile({
                filePath: join(__dirname, "..", "functions", "cloudfront", "url-rewrite.js")
            }),
            functionName: "static-site-url-rewrite",
            comment: "Rewrites URL paths to include `/index.html` to be compatible with S3 + CloudFront hosting"
        });;

        const siteDistribution = new CloudFront.Distribution(this, `distribution`, {
            certificate: cloudFrontCert,
            defaultRootObject: "index.html",
            domainNames: [webUrl],
            minimumProtocolVersion: CloudFront.SecurityPolicyProtocol.TLS_V1_2_2021,
            defaultBehavior: {
                // origin: new CFOrigins.HttpOrigin(staticSiteBucket.bucketWebsiteDomainName),
                origin: new CFOrigins.S3Origin(staticSiteBucket, {
                    originAccessIdentity,
                }),
                cachePolicy: new CloudFront.CachePolicy(this, "cache-policy", {
                    minTtl: cdk.Duration.minutes(30),
                    maxTtl: cdk.Duration.minutes(60),
                }),
                compress: true,
                allowedMethods: CloudFront.AllowedMethods.ALLOW_GET_HEAD_OPTIONS,
                viewerProtocolPolicy: CloudFront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
                functionAssociations: [
                    {
                        function: rewriteURIFunc,
                        eventType: CloudFront.FunctionEventType.VIEWER_REQUEST,
                    }
                ]
            },
            errorResponses:[
                {
                    httpStatus: 404,
                    responseHttpStatus: 404,
                    responsePagePath: '/404.html',
                    ttl: cdk.Duration.minutes(30),
                }
            ],
        });

        new Route53.ARecord(this, `site-record`, {
            target: Route53.RecordTarget.fromAlias(new Route53Targets.CloudFrontTarget(siteDistribution)),
            zone: hostedZone,
            recordName: webUrl,
        });

        new S3Deployment.BucketDeployment(this, `deployment`, {
            sources: [S3Deployment.Source.bucket(buildBucket, `${props.environmentName}.zip`)],
            destinationBucket: staticSiteBucket,
            prune: true,
            distribution: siteDistribution,
            destinationKeyPrefix: "/",
        });

        const dashboard = new CloudwatchDashboard(this, `dashboard`, {
            ...props,
            table: tableStack.table,
            queue: queues.queue,
            dlq: queues.dlq,
            serverLogGroup: serverLogGroup,
            workerLogGroup: workerLogGroup,
        });
    }
}