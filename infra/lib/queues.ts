import * as cdk from 'aws-cdk-lib';
import * as SQS from 'aws-cdk-lib/aws-sqs';
import { Construct } from 'constructs';
import { InfraStackProps } from './infra-stack';

/**
 * ClassificationQueue defines the queue from which the classification worker reads
 * Each inbox that should be classified should have its details enqueued here
 * to be classified when resources are ready
 */
export class ClassificationQueueStack extends cdk.Stack {
    public readonly queueArn: string;
    public readonly dlqArn: string;

    constructor(scope: Construct, id: string, props: InfraStackProps) {
      super(scope, id, props);
  
      const dlqName = `classify_dlq_${props.environmentName}.fifo`;
      const dlq = new SQS.Queue(this, dlqName, {
        queueName: dlqName,
        fifo: true,
      });
      this.dlqArn = dlq.queueArn;

      const queueName = `classify_queue_${props.environmentName}.fifo`;
      const queue = new SQS.Queue(this, queueName, {
        queueName,
        fifo: true,
        deadLetterQueue: {
          queue: dlq,
          maxReceiveCount: 3,
        },
      })
      this.queueArn = queue.queueArn;
    }
  }