import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import { InterestStack } from './interest';
import { ClassificationQueueStack } from './queues';
// import * as sqs from 'aws-cdk-lib/aws-sqs';

export type InfraStackProps = cdk.StackProps & {
  environmentName: string;
};

export class InfraStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props: InfraStackProps) {
    super(scope, id, props);

    const staticInterestSite = new InterestStack(this, `interest-site-${props?.environmentName}`, props);

    const queues = new ClassificationQueueStack(this, `classify-queues-${props.environmentName}`, props);
  }
}
