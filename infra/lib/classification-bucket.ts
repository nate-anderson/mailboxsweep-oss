import * as cdk from 'aws-cdk-lib';
import * as S3 from 'aws-cdk-lib/aws-s3';
import { Construct } from 'constructs';

export type ClassificationBucketStackProps = cdk.StackProps & {
    environmentName: string;
}

export class ClassificationBucketStack extends cdk.Stack {
    public readonly classificationBucket: S3.Bucket;

    constructor(scope: Construct, id: string, props: ClassificationBucketStackProps) {
        super(scope, id, props);

        const bucketName = `com.mailboxsweep.${props.environmentName}.classification`
        const bucket = new S3.Bucket(this, `classifications-${props.environmentName}`, {
            bucketName,
            encryption: S3.BucketEncryption.S3_MANAGED,
            removalPolicy: cdk.RemovalPolicy.RETAIN,
            lifecycleRules: [
                {
                    id: "ttl",
                    expiration: cdk.Duration.days(30),
                }
            ]
        });
        this.classificationBucket = bucket;
    }
}