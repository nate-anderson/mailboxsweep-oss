import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import { InterestStack } from './interest';
// import * as sqs from 'aws-cdk-lib/aws-sqs';

export type InfraStackProps = cdk.StackProps & {
  envirnomentName: string;
};

export class InfraStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props: InfraStackProps) {
    super(scope, id, props);

    const staticInterestSite = new InterestStack(this, `interest-site-${props?.envirnomentName}`, props)
  }
}
