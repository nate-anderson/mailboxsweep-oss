import * as cdk from 'aws-cdk-lib';
import * as S3 from 'aws-cdk-lib/aws-s3';
import { Construct } from 'constructs';
import { InfraStackProps } from './infra-stack';
import * as CloudFront from 'aws-cdk-lib/aws-cloudfront';

export default class FrontendBucketStack extends cdk.Stack {
    public buildBucket: S3.Bucket;
    public siteBucket: S3.Bucket;
    public siteOriginAccessIdentity: CloudFront.OriginAccessIdentity;

    constructor(scope: Construct, id: string, props: InfraStackProps) {
        super(scope, id, props);

        const buildBuckets = new S3.Bucket(this, `static-builds-bucket`, {
            bucketName: `com.mailboxsweep.builds.${props.environmentName}.web`,
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

        this.buildBucket = buildBuckets;

        const siteBucket = new S3.Bucket(this, `web-bucket`, {
            bucketName: `com.mailboxsweep.${props.environmentName}.web`,
            enforceSSL: true,
            // yes, this is a public static site and no sensitive information should be in this bucket
            // but access should be limited to be only through CloudFront to prevent excessive downloads
            // racking up our bill
            publicReadAccess: false,
            removalPolicy: cdk.RemovalPolicy.RETAIN,
            websiteIndexDocument: "index.html",
            websiteErrorDocument: "404.html"
        });

        this.siteBucket = siteBucket;

        const siteOriginAccessIdentity = new CloudFront.OriginAccessIdentity(this, `site-${props.environmentName}-origin`);
        siteBucket.grantRead(siteOriginAccessIdentity);

        this.siteOriginAccessIdentity = siteOriginAccessIdentity;
    }
}