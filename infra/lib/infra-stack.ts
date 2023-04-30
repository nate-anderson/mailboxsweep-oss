import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import { AppStack } from './app';
import { RepositoryStack } from './ecr';
import { InterestStack } from './interest';

export type InfraStackProps = cdk.StackProps & {
  environmentName: string;
};

export class InfraStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props: InfraStackProps) {
    super(scope, id, props);

    const staticInterestSite = new InterestStack(this, `interest-site`, props);
    
    const ecrRepos = new RepositoryStack(this, `ecr-repos`, props);
  }
}
