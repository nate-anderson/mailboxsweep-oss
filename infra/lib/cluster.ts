import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import * as ECS from 'aws-cdk-lib/aws-ecs';
import { InfraStackProps } from './infra-stack';
import * as EC2 from 'aws-cdk-lib/aws-ec2';

type ClusterProps = InfraStackProps & {
  vpc: EC2.Vpc;
  instanceType: string;
};

export class Cluster extends cdk.Stack {
    constructor(scope: Construct, id: string, props: ClusterProps) {
      super(scope, id, props);
  
      const clusterId = `cluster-${props.environmentName}`;
      const cluster = new ECS.Cluster(this, clusterId, {
        clusterName: clusterId,
        vpc: props?.vpc,
        capacity: {
          instanceType: new EC2.InstanceType(props.instanceType),
        },
      }); 
    }
}