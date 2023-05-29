import * as cdk from 'aws-cdk-lib';
import * as ECR from 'aws-cdk-lib/aws-ecr';
import { Construct } from 'constructs';
import { InfraStackProps } from './infra-stack';

export class RepositoryStack extends cdk.Stack {
    public serverRepoArn: string;
    public serverRepo: ECR.Repository;
    public workerRepoArn: string;
    public workerRepo: ECR.Repository;

    constructor(scope: Construct, id: string, props: InfraStackProps) {
        super(scope, id, props);

        const serverRepoName = `mailboxsweep-server-${props.environmentName}`;
        const serverRepo = new ECR.Repository(this, serverRepoName, {
            repositoryName: serverRepoName,
            lifecycleRules: [
                {
                    description: "Retain last 5 versions",
                    maxImageCount: 5,
                    tagStatus: ECR.TagStatus.ANY,
                    rulePriority: 1,
                }
            ],
            removalPolicy: cdk.RemovalPolicy.RETAIN,
        });
        this.serverRepo = serverRepo;
        this.serverRepoArn = serverRepo.repositoryArn;

        const workerRepoName = `mailboxsweep-worker-${props.environmentName}`;
        const workerRepo = new ECR.Repository(this, workerRepoName, {
            repositoryName: workerRepoName,
            lifecycleRules: [
                {
                    description: "Retain last 5 versions",
                    maxImageCount: 5,
                    tagStatus: ECR.TagStatus.ANY,
                    rulePriority: 1,
                }
            ],
            removalPolicy: cdk.RemovalPolicy.RETAIN,
        });
        this.workerRepo = serverRepo;
        this.workerRepoArn = serverRepo.repositoryArn;
    }
}