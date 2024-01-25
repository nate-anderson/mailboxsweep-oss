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
import { ClassificationQueueStack, CommitChangesetQueueStack } from './queues';
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
import { CloudwatchDashboardStack } from './cloudwatch-dashboard';
import * as SES from "aws-cdk-lib/aws-ses";
import * as SNS from "aws-cdk-lib/aws-sns";
import * as SNSSubscriptions from "aws-cdk-lib/aws-sns-subscriptions";
import * as WAF from 'aws-cdk-lib/aws-wafv2';
import { join } from 'path';
import { AlarmsStack } from './alarms';

type AppProps = InfraStackProps & {
    subdomain?: string;
    clusterInstanceType: string;
    clusterInstanceCount: number;
    xrayEnabled: boolean;
};

const Port = {
    HTTPS: 443,
    HTTP: 80
}

const errorEmail = "tech@mailboxsweep.com";

export class AppStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props: AppProps) {
        super(scope, id, props);

        const tableStack = new TableStack(this, `app-table-${props.environmentName}`, props);
        const classifyQueues = new ClassificationQueueStack(this, `classify-queues-${props.environmentName}`, props);
        const commitChangesetQueues = new CommitChangesetQueueStack(this, `changeset-queues-${props.environmentName}`, props);

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

        const vpc = new Vpc(this, `vpc-${props.environmentName}`, {
            // enableDnsSupport: true,
            // enableDnsHostnames: true,
        });

        const clusterId = `cluster-${props.environmentName}`;
        const cluster = new ECS.Cluster(this, `mailboxsweep-cluster`, {
            clusterName: clusterId,
            vpc,
            capacity: {
              instanceType: new EC2.InstanceType(props.clusterInstanceType),
            },
            containerInsights: true,
        });

        /**
         * @TODO retry deployment with DNS namespace
         * didn't try this last night
         */
        // const namespace = 'cluster.mailboxsweep.com';
        // const dnsNamespace = cluster.addDefaultCloudMapNamespace({
        //     vpc,
        //     name: namespace,
        //     type: ServiceDiscovery.NamespaceType.DNS_PRIVATE,
        // });
        

        // current cluster doesn't have enough capacity, try with more capacity
        if (props.clusterInstanceCount > 1) {
            for (let i = 1; i < props.clusterInstanceCount; i++) {
                cluster.addCapacity(`cap-${i}`, {
                    instanceType: new EC2.InstanceType(props.clusterInstanceType),
                });
            }
        }

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
                    resources: [classifyQueues.queue.queueArn, commitChangesetQueues.queue.queueArn],
                })
            ]
        })

        serverTaskDefinition.taskRole.attachInlinePolicy(readSSMPolicy);
        serverTaskDefinition.taskRole.attachInlinePolicy(serverWriteQueuePolicy);
        serverTaskDefinition.taskRole.addManagedPolicy({
            managedPolicyArn: 'arn:aws:iam::aws:policy/AWSXRayDaemonWriteAccess',
        });

        const webUrl = Resources[props.environmentName].web.dns.fullyQualifiedDomainName;

        const XRayConfig = {
            port: 2000,
            containerName: 'xray-sidecar',
        };

        const appConfig: Record<string, string> = {
            // Generic app config
            ENV: props.environmentName,
            HTTP_PORT: Port.HTTP.toString(),

            DEV_MODE: "false",
            CHANGESET_PAGE_ID_HASH_MOD: "11",
            INBOX_METADATA_TTL_MINUTES: "30",
            CLASSIFICATION_PAGE_SIZE: "30",
            CORS_ALLOWED_ORIGINS: `https://${webUrl}`,
            MINUTES_BETWEEN_OPEN_REQUESTS: "120",
            MINUTES_BETWEEN_CLOSED_REQUESTS: "10",

            // Login
            BASE_URL: `https://${webUrl}/login/token`,
            CLASSIFY_SELECT_URL: `https://${webUrl}/changeset/select`,

            // DB
            DYNAMODB_REGION: "us-west-2",
            DYNAMODB_TABLE: tableStack.table.tableName,

            // Bucket
            S3_REGION: "us-west-2",
            S3_BUCKET: bucket.bucketName,

            // Queues
            SQS_REGION: "us-west-2",
            
            SQS_CLASSIFY_QUEUE_URL: classifyQueues.queue.queueUrl,
            SQS_CLASSIFY_QUEUE: classifyQueues.queue.queueName,
            SQS_CLASSIFY_DLQ_URL: classifyQueues.dlq.queueUrl,
            SQS_CLASSIFY_DLQ: classifyQueues.dlq.queueName,

            SQS_COMMIT_QUEUE_URL: commitChangesetQueues.queue.queueUrl,
            SQS_COMMIT_QUEUE: commitChangesetQueues.queue.queueName,
            SQS_COMMIT_DLQ_URL: commitChangesetQueues.dlq.queueUrl,
            SQS_COMMIT_DLQ: commitChangesetQueues.dlq.queueName,
            

            SES_REGION: "us-west-2",
            SES_LOGIN_EMAIL_TEMPLATE_NAME: "mailboxsweep_login_link",
            SES_CLASSIFY_COMPLETE_EMAIL_TEMPLATE_NAME: "mailboxsweep_classify_complete",

            // Sessions & cookies
            COOKIE_DOMAIN: webUrl,
            SESSION_KEY_LENGTH: "30",
            COOKIE_PATH: "/",
            LOGIN_LINK_DURATION_MINUTES: "10",
            SESSION_DURATION_MINUTES: "1440",

            // Stripe
            PAYMENT_ACCESS_DURATION_DAYS: "30",
            STRIPE_PAYMENT_CONFIRMATION_URL: `https://${webUrl}/plan/payment-success?checkoutSessionId={CHECKOUT_SESSION_ID}`,
            STRIPE_CHECKOUT_SESSION_DURATION_MINUTES: "30",

            GOOGLE_OAUTH_REDIRECT_URL: `https://${webUrl}/login/gmail`,
            SES_CONFIGURATION_SET_NAME: "MailboxSweep_Transactional",

            // AWS_XRAY_DAEMON_ADDRESS: `${XRayConfig.containerName}.${namespace}:${XRayConfig.port}`,
            AWS_XRAY_ENABLED: props.xrayEnabled.toString(),

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
                FORCE_BUILD_REFRESH: "true",
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
                interval: cdk.Duration.seconds(120),  
            },
        });

        // server must be able to read and serve classification trees
        bucket.grantRead(serverTaskDefinition.taskRole);

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
                FORCE_BUILD_REFRESH: "true",
            },
            memoryReservationMiB: 1024,
            logging: ECS.LogDrivers.awsLogs({
                streamPrefix: "mailboxsweep-worker",
                logGroup: workerLogGroup,
            }),
            portMappings: [
                {
                    containerPort: XRayConfig.port,
                }
            ],
        });
        workerTaskDefinition.taskRole.attachInlinePolicy(readSSMPolicy);
        workerTaskDefinition.taskRole.addManagedPolicy({
            managedPolicyArn: 'arn:aws:iam::aws:policy/AWSXRayDaemonWriteAccess',
        });

        const workerServiceId = `worker-service`;
        const workerService = new ECS.Ec2Service(this, workerServiceId, {
            cluster,
            taskDefinition: workerTaskDefinition,
            desiredCount: 1,
            // cloudMapOptions: {
            //     cloudMapNamespace: dnsNamespace,
            // },
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
                        "sqs:DeleteMessage",
                        "sqs:ChangeMessageVisibility",
                    ],
                    resources: [classifyQueues.queue.queueArn, classifyQueues.dlq.queueArn, commitChangesetQueues.queue.queueArn, commitChangesetQueues.dlq.queueArn],
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
                    resources: [classifyQueues.dlq.queueArn, commitChangesetQueues.dlq.queueArn],
                })
            ]
        });
        workerTaskDefinition.taskRole.attachInlinePolicy(readQueuePolicy);
        workerTaskDefinition.taskRole.attachInlinePolicy(writeDLQPolicy);
        workerTaskDefinition.taskRole.attachInlinePolicy(sendEmailPolicy);

        // worker must read classification leaves when modifying
        bucket.grantRead(workerTaskDefinition.taskRole);
 
        if (props.xrayEnabled) {
            const xraySidecarTaskDefinition = new ECS.TaskDefinition(this, `xray-sidecar`, {
                compatibility: ECS.Compatibility.EC2,
            });
            
            xraySidecarTaskDefinition.addContainer('xray-sidecar-task', {
                containerName: XRayConfig.containerName,
                image: ContainerImage.fromRegistry("amazon/aws-xray-daemon"),
                cpu: 32,
                memoryReservationMiB: 256,
                portMappings: [
                    {
                        hostPort: XRayConfig.port,
                        containerPort: XRayConfig.port,
                        protocol: ECS.Protocol.UDP,
                    }
                ]
            });
            const xraySidecarService = new ECS.Ec2Service(this, `xray-sidecar-service`, {
                taskDefinition: xraySidecarTaskDefinition,
                cluster,
                desiredCount: 1,
                // cloudMapOptions: {
                //     cloudMapNamespace: dnsNamespace,
                //     name: XRayConfig.containerName,
                //     containerPort: XRayConfig.port,
                //     dnsRecordType: ServiceDiscovery.DnsRecordType.SRV,
                // },
            });
        }


        const cert = CertMan.Certificate.fromCertificateArn(this, `api-cert-${props.environmentName}`, Resources.global.api.cert.arn);

        const securityGroup = new EC2.SecurityGroup(this, `api-security-group`, {
            vpc,
        });
        securityGroup.addIngressRule(EC2.Peer.ipv4('0.0.0.0/0'), EC2.Port.tcp(Port.HTTP));

        const managedRules: WAF.CfnWebACL.RuleProperty[] = [
            // AWS IP Reputation list includes known malicious actors/bots and is regularly updated    
            {
                name: 'AWS-AWSManagedRulesAmazonIpReputationList',
                priority: 10,
                statement: {
                    managedRuleGroupStatement: {
                        vendorName: 'AWS',
                        name: 'AWSManagedRulesAmazonIpReputationList',
                    },
                },
                overrideAction: {
                    none: {},
                },
                visibilityConfig: {
                    sampledRequestsEnabled: true,
                    cloudWatchMetricsEnabled: true,
                    metricName: 'AWSManagedRulesAmazonIpReputationList',
                },
            },
            
            // Common Rule Set aligns with major portions of OWASP Core Rule Set    
            {
                name: 'AWS-AWSManagedRulesCommonRuleSet',
                priority: 20,
                statement: {
                    managedRuleGroupStatement: {
                    vendorName: 'AWS',
                    name: 'AWSManagedRulesCommonRuleSet',
                    // Excluding generic RFI body rule for sns notifications
                    // https://docs.aws.amazon.com/waf/latest/developerguide/aws-managed-rule-groups-list.html
                      excludedRules: [
                       { name: 'GenericRFI_BODY' },
                       { name: 'SizeRestrictions_BODY' },
                       { name: 'SizeRestrictions_URIPATH' },
                    ],
                    },
                },
                overrideAction: {
                    none: {},
                },
                visibilityConfig: {
                    sampledRequestsEnabled: true,
                    cloudWatchMetricsEnabled: true,
                    metricName: 'AWS-AWSManagedRulesCommonRuleSet',
                },
            },
            // Blocks common SQL Injection
            {
                name: 'AWSManagedRulesSQLiRuleSet',
                priority: 30,
                visibilityConfig: {
                    sampledRequestsEnabled: true,
                    cloudWatchMetricsEnabled: true,
                    metricName: 'AWSManagedRulesSQLiRuleSet',
                },
                overrideAction: {
                    none: {},
                },
                statement: {
                    managedRuleGroupStatement: {
                    vendorName: 'AWS',
                    name: 'AWSManagedRulesSQLiRuleSet',
                    excludedRules: [],
                    },
                },
            },
            // Blocks attacks targeting LFI(Local File Injection) for linux systems
            {
                name: 'AWSManagedRuleLinux',
                priority: 50,
                visibilityConfig: {
                    sampledRequestsEnabled: true,
                    cloudWatchMetricsEnabled: true,
                    metricName: 'AWSManagedRuleLinux',
                },
                overrideAction: {
                    none: {},
                },
                statement: {
                    managedRuleGroupStatement: {
                        vendorName: 'AWS',
                        name: 'AWSManagedRulesLinuxRuleSet',
                        excludedRules: [],
                    },
                },
            },

            // Blocks bot interations
            {
                name: 'AWS-BotControl',
                priority: 60,
                statement: {
                    managedRuleGroupStatement: {
                        vendorName: "AWS",
                        name: "AWSManagedRulesBotControlRuleSet",
                        excludedRules: [],
                    },
                },
                visibilityConfig: {
                    sampledRequestsEnabled: true,
                    cloudWatchMetricsEnabled: true,
                    metricName: "AWS-AWSBotControl",
                },
                overrideAction: {
                    none: {},
                },
            },
            {
                name: 'RateLimit',
                priority: 70,
                action: {
                    block: {},
                },
                statement: {
                    rateBasedStatement: {
                        limit: 200,
                        aggregateKeyType: "IP",
                    }
                },
                visibilityConfig: {
                    sampledRequestsEnabled: true,
                    cloudWatchMetricsEnabled: true,
                    metricName: "LimitRequests200"
                }
            }
        ];

        const loadBalancedAPIService = new ECSPatterns.ApplicationLoadBalancedEc2Service(this, `server-service`, {
            cluster,
            taskDefinition: serverTaskDefinition,
            cpu: 256,
            memoryReservationMiB: 512,
            desiredCount: 2,
            protocol: LoadBalancingV2.ApplicationProtocol.HTTPS,
            openListener: true,
            domainZone: hostedZone,
            certificate: cert,
            publicLoadBalancer: true,
            maxHealthyPercent: 200,
            minHealthyPercent: 100,
            // cloudMapOptions: {
            //     cloudMapNamespace: dnsNamespace,
            //     name: `server`,
            //     containerPort: Port.HTTP,
            //     dnsRecordType: ServiceDiscovery.DnsRecordType.SRV,
            // },
        });
        loadBalancedAPIService.targetGroup.configureHealthCheck({
            path: "/healthcheck",
            port: 'traffic-port',
            protocol: LoadBalancingV2.Protocol.HTTP,
            interval: cdk.Duration.seconds(120),
            healthyThresholdCount: 2,
            unhealthyThresholdCount: 2,
        });
        const webAcl = new WAF.CfnWebACL(this, `api-rate-limit`, {
            defaultAction: {
                allow: {
                     
                }
            },
            visibilityConfig: {
                cloudWatchMetricsEnabled: true,
                metricName: `${props.environmentName}-waf`,
                sampledRequestsEnabled: false,
            },
            scope: 'REGIONAL',
            name: `waf-${props.environmentName}`,
            rules: managedRules,
        });

        new WAF.CfnWebACLAssociation(this, `web-api-acl`, {
            resourceArn: loadBalancedAPIService.loadBalancer.loadBalancerArn,
            webAclArn: webAcl.attrArn,
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
                    enableAcceptEncodingBrotli: true,
                    enableAcceptEncodingGzip: true,
                }),
                compress: true,
                allowedMethods: CloudFront.AllowedMethods.ALLOW_GET_HEAD_OPTIONS,
                viewerProtocolPolicy: CloudFront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
                functionAssociations: [
                    {
                        function: rewriteURIFunc,
                        eventType: CloudFront.FunctionEventType.VIEWER_REQUEST,
                    }
                ],
                responseHeadersPolicy: new CloudFront.ResponseHeadersPolicy(this, "add-anti-clickjacking-headers", {
                    responseHeadersPolicyName: "add-anti-clickjacking-headers",
                    securityHeadersBehavior: {
                        frameOptions: {
                            override: true,
                            frameOption: CloudFront.HeadersFrameOption.DENY,
                        },
                        contentSecurityPolicy: {
                            override: true,
                            contentSecurityPolicy: "frame-ancestors 'none'",
                        }
                    }
                }),
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

        const dashboard = new CloudwatchDashboardStack(this, `dashboard`, {
            ...props,
            table: tableStack.table,
            queue: classifyQueues.queue,
            dlq: classifyQueues.dlq,
            serverLogGroup: serverLogGroup,
            workerLogGroup: workerLogGroup,
            loadBalancer: loadBalancedAPIService.loadBalancer,
        });

        // SES configuration
        const sesConfigSet = new SES.ConfigurationSet(this, `mailboxsweep_transactional`, {
            configurationSetName: "MailboxSweep_Transactional",
            reputationMetrics: true,
            suppressionReasons: SES.SuppressionReasons.BOUNCES_AND_COMPLAINTS,
        });

        // send notifications to tech@mailboxsweep.com when render fails, a customer complains, an email bounces or is rejected
        const sesRenderFailureTopic = new SNS.Topic(this, `mailboxsweep_ses_errors`, {
            fifo: false,
            topicName: "SESErrors",
            displayName: "MailboxSweep SES Failures",
        });
        sesRenderFailureTopic.addSubscription(new SNSSubscriptions.EmailSubscription(errorEmail, {}));
        
        new SES.CfnConfigurationSetEventDestination(this, `ses-error-dest`, {
            configurationSetName: sesConfigSet.configurationSetName,
            eventDestination: {
                matchingEventTypes: ['renderingFailure', 'complaint', 'bounce', 'reject'],
                enabled: true,
                snsDestination: {
                    topicArn: sesRenderFailureTopic.topicArn,
                }
            }
        })

        const alarms = new AlarmsStack(this, `alarms`, {
            ...props,
            emailAddr: "tech@mailboxsweep.com",
            commitQueue: commitChangesetQueues.queue,
            commitDLQ: commitChangesetQueues.dlq,
            classifyQueue: classifyQueues.queue,
            classifyDLQ: classifyQueues.dlq,
            loadBalancer: loadBalancedAPIService.loadBalancer,
        });
    }
}