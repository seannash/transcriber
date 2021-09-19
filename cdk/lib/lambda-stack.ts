import * as lambda from '@aws-cdk/aws-lambda-go'
import * as cdk from '@aws-cdk/core'
import * as s3 from '@aws-cdk/aws-s3'
import * as ec2 from '@aws-cdk/aws-ec2'
import * as ecs from '@aws-cdk/aws-ecs'
import * as lambdaEventSources from '@aws-cdk/aws-lambda-event-sources'
import * as ecsPatterns from '@aws-cdk/aws-ecs-patterns'
import * as events from '@aws-cdk/aws-events'
import * as targets from '@aws-cdk/aws-events-targets'
import * as sqs from '@aws-cdk/aws-sqs'
import * as ddb from '@aws-cdk/aws-dynamodb'
import * as cognito from '@aws-cdk/aws-cognito'
import { UserPoolClient } from '@aws-cdk/aws-cognito'
import { ManagedPolicy, PolicyStatement, Effect} from '@aws-cdk/aws-iam'
import { Queue } from '@aws-cdk/aws-sqs'
import { AwsLogDriver } from '@aws-cdk/aws-ecs'
import { removeAllListeners } from 'process'
import { countResources } from '@aws-cdk/assert'

export interface LambdaStackProps extends cdk.StackProps {
  readonly bucket: s3.Bucket
  readonly cluster: ecs.Cluster;
  readonly jobTable: ddb.Table;
  readonly queue: sqs.Queue;
  readonly userPool: cognito.UserPool;
}

const sendingEmail: string = "sean.c.nash@gmail.com"

export class LambdaStack extends cdk.Stack {
  constructor(scope: cdk.App, id: string, props: LambdaStackProps) {
    super(scope, id, props);

    this.setupSendEmailLambda(props);
    this.setupTranscriberFinishLambda(props);
  }  

  setupTranscriberFinishLambda(props: LambdaStackProps) {
    const transcriberFinish = new lambda.GoFunction(this, 'TranscriberFinish', {
      entry: '../cmd/TranscriberFinishLambda/',
      environment: {
        'EMAIL_USER': sendingEmail,
        'USER_POOL': props.userPool.userPoolArn,
        'TABLE_NAME': props.jobTable.tableName,
        'EMAIL_QUEUE_URL': props.queue.queueUrl
      },
    })
    props.jobTable.grantReadWriteData(transcriberFinish)
    props.queue.grantSendMessages(transcriberFinish)
    const finishRule = new events.Rule(this, 'FinishRule', {
      ruleName: 'FinishRule',
      eventPattern: {
        source: [ 'aws.transcribe' ],
        detailType: [ 'Transcribe Job State Change' ]
      },
      targets: [ new targets.LambdaFunction(transcriberFinish) ]
    })
  }

  setupSendEmailLambda(props: LambdaStackProps) {
    const sendEmailLambda = new lambda.GoFunction(this, 'SendEmailLambda', {
      entry: '../cmd/SendEmailLambda/',
      environment: {
        SENDING_EMAIL: sendingEmail,
        USER_POOL: props.userPool.userPoolId
      }
    })

    sendEmailLambda.addToRolePolicy(new PolicyStatement({
      actions: ['ses:SendEmail', 'SES:SendRawEmail'],
      resources: ['*'],
      effect: Effect.ALLOW,
    }));

    sendEmailLambda.addToRolePolicy(new PolicyStatement({
      actions: ['cognito-idp:AdminGetUser'],
      resources: [props.userPool.userPoolArn],
      effect: Effect.ALLOW,
    }));
    
    sendEmailLambda.addEventSource(new lambdaEventSources.SqsEventSource(props.queue, {
      batchSize: 1,
      maxBatchingWindow: cdk.Duration.minutes(1),
    }));

  }

}
