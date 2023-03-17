import * as cdk from 'aws-cdk-lib';
import * as ECR from 'aws-cdk-lib/aws-ecr';
import { Construct } from 'constructs';
import { InfraStackProps } from './infra-stack';

export class RepositoryStack extends cdk.Stack {
    public repoArn: string;

    constructor(scope: Construct, id: string, props: InfraStackProps) {
        super(scope, id, props);
        const repoName = `mailboxsweep-repo-${props.environmentName}`;

        const repo = new ECR.Repository(this, repoName, {
            repositoryName: repoName,
            lifecycleRules: [
                {
                    description: "Retain last 5 versions",
                    maxImageCount: 5,
                    tagStatus: ECR.TagStatus.ANY,
                    rulePriority: 1,
                }
            ],
            removalPolicy: cdk.RemovalPolicy.RETAIN,
            imageTagMutability: ECR.TagMutability.IMMUTABLE,
        });

        this.repoArn = repo.repositoryArn;
    }
}