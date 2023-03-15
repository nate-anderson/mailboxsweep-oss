import * as cdk from 'aws-cdk-lib';
import * as S3 from 'aws-cdk-lib/aws-s3';
import * as CloudFront from 'aws-cdk-lib/aws-cloudfront';
import * as S3Deployment from 'aws-cdk-lib/aws-s3-deployment';
import { Construct } from 'constructs';
import { resolve } from "path";
import * as route53 from 'aws-cdk-lib/aws-route53';
import * as acm from 'aws-cdk-lib/aws-certificatemanager';
import * as targets from 'aws-cdk-lib/aws-route53-targets';
import * as CFOrigins from 'aws-cdk-lib/aws-cloudfront-origins';
import { InfraStackProps } from './infra-stack';

/**
 * Static site with signup form for interest newsletter using Mailchimp
 */

export class InterestStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props: InfraStackProps) {
        super(scope, id, props);

        const bucketName = `mailboxsweep-interest-${props.envirnomentName}-bucket`;
        const siteBucket = new S3.Bucket(this, bucketName, {
            bucketName: bucketName,
            publicReadAccess: true,
            removalPolicy: cdk.RemovalPolicy.DESTROY,
            // just a static site that is in VCS, blow it up
            autoDeleteObjects: true,
        });

        const zone = route53.HostedZone.fromLookup(this, "Zone", {
            domainName: "mailboxsweep.com",
        });

        const siteCert = new acm.DnsValidatedCertificate(this, "SiteCertificate", {
            domainName: "mailboxsweep.com",
            hostedZone: zone,
            region: "us-east-1",  //standard for acm certs
        });
        

        const deploymentName = `mailboxsweep_interest_${props.envirnomentName}_bdpl`;
        const deployment = new S3Deployment.BucketDeployment(this, deploymentName, {
            destinationBucket: siteBucket,
            sources: [S3Deployment.Source.asset(resolve(__dirname, "../../interest"))]
        })
        const originAccessIdentity = new CloudFront.OriginAccessIdentity(this, 'OriginAccessIdentity');

        const distName = `mailboxsweep_interest_dist_${props.envirnomentName}_dist`;
        const siteDistribution = new CloudFront.Distribution(this, distName, {
            certificate: siteCert,
            defaultRootObject: "index.html",
            domainNames: ["mailboxsweep.com"],
            minimumProtocolVersion: CloudFront.SecurityPolicyProtocol.TLS_V1_2_2021,
            errorResponses:[
                {
                httpStatus: 403,
                responseHttpStatus: 403,
                responsePagePath: '/error.html',
                ttl: cdk.Duration.minutes(30),
                }
            ],
            defaultBehavior: {
                origin: new CFOrigins.S3Origin(siteBucket, {originAccessIdentity: originAccessIdentity}),
                compress: true,
                allowedMethods: CloudFront.AllowedMethods.ALLOW_GET_HEAD_OPTIONS,
                viewerProtocolPolicy: CloudFront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
      }
        });
        
        const recordName = `mailboxsweep_interest_arecord_${props.envirnomentName}`;
        new route53.ARecord(this, recordName, {
            target: route53.RecordTarget.fromAlias(new targets.CloudFrontTarget(siteDistribution)),
            zone,
        });
    }
}