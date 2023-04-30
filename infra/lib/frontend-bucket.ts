import * as cdk from 'aws-cdk-lib';
import * as S3 from 'aws-cdk-lib/aws-s3';
import { Construct } from 'constructs';
import { InfraStackProps } from './infra-stack';

export default class FrontendBucketStack extends cdk.Stack {
    public bucket: S3.Bucket;

    constructor(scope: Construct, id: string, props: InfraStackProps) {
        super(scope, id, props);

        const staticSiteBucket = new S3.Bucket(this, `static-site-bucket`, {
            bucketName: `com.mailboxsweep.${props.environmentName}.web`,
            enforceSSL: true,
            removalPolicy: cdk.RemovalPolicy.RETAIN,
            versioned: true,
            lifecycleRules: [
                {
                    id: `retain-latest-5-versions`,
                    enabled: true,
                    noncurrentVersionsToRetain: 4,
                    // keep old versions for one week
                    noncurrentVersionExpiration: cdk.Duration.days(7),
                },
            ]
        });

        this.bucket = staticSiteBucket;
    }
}