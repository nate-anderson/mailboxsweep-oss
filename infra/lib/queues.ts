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
  constructor(scope: Construct, id: string, props: InfraStackProps) {
    super(scope, id, props);

    const dlqName = `classify_dlq_${props.environmentName}.fifo`;
    const dlq = new SQS.Queue(this, dlqName, {
      queueName: dlqName,
      fifo: true,
      removalPolicy: cdk.RemovalPolicy.RETAIN,
    });
    this.exportValue(dlq.queueArn, { name: `classifydlq-arn` });

    const queueName = `classify_queue_${props.environmentName}.fifo`;
    const queue = new SQS.Queue(this, queueName, {
      queueName,
      fifo: true,
      deadLetterQueue: {
        queue: dlq,
        maxReceiveCount: 3,
      },
      removalPolicy: cdk.RemovalPolicy.RETAIN,
    });
    this.exportValue(queue.queueArn, { name: `classify-queue-arn` });

    const imapCooloffDlqName = `imap_cooloff_dlq_${props.environmentName}.fifo`;
    const imapCooloffDlq = new SQS.Queue(this, imapCooloffDlqName, {
      queueName: imapCooloffDlqName,
      fifo: true,
      removalPolicy: cdk.RemovalPolicy.RETAIN,
    });
    this.exportValue(imapCooloffDlq.queueArn, { name: `imap-cooloff-dlq` });

    const imapCooloffQueueName = `imap_cooloff_queue_${props.environmentName}.fifo`;
    const imapCooloffQueue = new SQS.Queue(this, imapCooloffQueueName, {
      queueName: imapCooloffQueueName,
      fifo: true,
      deadLetterQueue: {
        queue: imapCooloffDlq,
        maxReceiveCount: 3,
      },
      removalPolicy: cdk.RemovalPolicy.RETAIN,
    });
    this.exportValue(imapCooloffQueue.queueArn, { name: `imap-cooloff-queue` });

  }
}

export class CommitChangesetQueueStack extends cdk.Stack {

  constructor(scope: Construct, id: string, props: InfraStackProps) {
    super(scope, id)

    const dlqName = `commit_changeset_dlq_${props.environmentName}.fifo`;
    const dlq = new SQS.Queue(this, dlqName, {
      queueName: dlqName,
      fifo: true,
      removalPolicy: cdk.RemovalPolicy.RETAIN,
    });
    this.exportValue(dlq.queueArn, { name: `commit-changeset-dlq-arn` });

    const queueName = `commit_changeset_queue_${props.environmentName}.fifo`;
    const queue = new SQS.Queue(this, queueName, {
      queueName,
      fifo: true,
      deadLetterQueue: {
        queue: dlq,
        maxReceiveCount: 3,
      },
      removalPolicy: cdk.RemovalPolicy.RETAIN,
    });

    this.exportValue(queue.queueArn, { name: `commit-changeset-queue-arn` });
  }
}

export class QueuesStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props: InfraStackProps) {
    super(scope, id, props);

    new ClassificationQueueStack(this, `classify-queues-${props.environmentName}`, props);
    new CommitChangesetQueueStack(this, `changeset-queues-${props.environmentName}`, props);
  }
}
