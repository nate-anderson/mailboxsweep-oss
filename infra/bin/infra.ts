#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { InfraStack } from '../lib/infra-stack';
import { AppStack } from '../lib/app';
import { TableStack } from '../lib/table';
import { ClassificationBucketStack } from '../lib/classification-bucket';
import { QueuesStack } from '../lib/queues';

const environments = ["staging", "prod"];

const app = new cdk.App();
const infraStackStaging = new InfraStack(app, 'infra-staging', {
  environmentName: "staging",
  env: {
    region: "us-west-2",
    account: "968574982400",
  }
  /* If you don't specify 'env', this stack will be environment-agnostic.
   * Account/Region-dependent features and context lookups will not work,
   * but a single synthesized template can be deployed anywhere. */

  /* Uncomment the next line to specialize this stack for the AWS Account
   * and Region that are implied by the current CLI configuration. */
  // env: { account: process.env.CDK_DEFAULT_ACCOUNT, region: process.env.CDK_DEFAULT_REGION },

  /* Uncomment the next line if you know exactly what Account and Region you
   * want to deploy the stack to. */
  // env: { account: '123456789012', region: 'us-east-1' },

  /* For more information, see https://docs.aws.amazon.com/cdk/latest/guide/environments.html */
});

const infraStackProd = new InfraStack(app, 'infra-prod', {
  environmentName: "prod",
  env: {
    region: "us-west-2",
    account: "968574982400",
  }
});

const stagingTableStack = new TableStack(app, `table-staging`, {
  environmentName: "staging",
});

const stagingBucketStack = new ClassificationBucketStack(app, `classification-bucket-staging`, {
  environmentName: "staging"
});

const queues = new QueuesStack(app, `queues-stack-staging`, {
  environmentName: "staging",
});

new AppStack(app, `app-staging`, {
  environmentName: "staging",
  clusterInstanceType: "t2.medium",
  clusterInstanceCount: 1,
  maxClusterInstanceCount: 2,
  xrayEnabled: false,
  table: stagingTableStack.table,
  classificationBucket: stagingBucketStack.classificationBucket,

  classifyQueueArn: "arn:aws:sqs:us-west-2:968574982400:classify_queue_staging.fifo",
  classifyDlqArn: "arn:aws:sqs:us-west-2:968574982400:classify_dlq_staging.fifo",
  changesetQueueArn: "arn:aws:sqs:us-west-2:968574982400:commit_changeset_queue_staging.fifo",
  changesetDlqArn: "arn:aws:sqs:us-west-2:968574982400:commit_changeset_dlq_staging.fifo"
});

const prodPermanentStack = new TableStack(app, `table-prod`, {
  environmentName: "prod",
});

const prodBucketStack = new ClassificationBucketStack(app, `classification-bucket-prod`, {
  environmentName: "prod"
});

const prodQueuesStack = new QueuesStack(app, `queues-stack-prod`, {
  environmentName: "prod",
});

new AppStack(app, `app-prod`, {
  environmentName: "prod",
  clusterInstanceType: "t2.medium",
  clusterInstanceCount: 1,
  maxClusterInstanceCount: 5,
  xrayEnabled: false,
  table: prodPermanentStack.table,
  classificationBucket: prodBucketStack.classificationBucket,

  classifyQueueArn: "arn:aws:sqs:us-west-2:968574982400:classify_queue_prod.fifo",
  classifyDlqArn: "arn:aws:sqs:us-west-2:968574982400:classify_dlq_prod.fifo",
  changesetQueueArn: "arn:aws:sqs:us-west-2:968574982400:commit_changeset_queue_prod.fifo",
  changesetDlqArn: "arn:aws:sqs:us-west-2:968574982400:commit_changeset_dlq_prod.fifo"
});
