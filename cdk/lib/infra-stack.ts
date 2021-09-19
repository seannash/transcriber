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
import { Table } from '@aws-cdk/aws-dynamodb'

export class InfraStack extends cdk.Stack {
  readonly bucket: s3.Bucket
  readonly cluster: ecs.Cluster;
  readonly jobTable: ddb.Table;
  readonly queue: sqs.Queue;
  readonly userPool: cognito.UserPool;

  constructor(scope: cdk.App, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // Setup Cognito
    this.userPool = new cognito.UserPool(this, 'UserPool', {
      autoVerify: {
        email: true
      },
      userPoolName: 'TranscriberUserPool',
      signInCaseSensitive: true,
      passwordPolicy: {
        minLength: 6,
        requireLowercase: true,
        requireDigits: false,
        requireSymbols: false,
        requireUppercase: true
      }
    })
    const userPoolClient = new UserPoolClient(this, 'TranscriberUserPoolClient', {
      userPool: this.userPool,
      generateSecret: false,
      authFlows: {
        userPassword: true
      }
    }) 
    
    // Setup Ddb Table
    this.jobTable = new ddb.Table(this, 'Job', {
      billingMode: ddb.BillingMode.PROVISIONED,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
      partitionKey: {name: 'job', type: ddb.AttributeType.STRING}, 
    })
    this.jobTable.addGlobalSecondaryIndex({
      indexName: 'user-index',
      partitionKey:  {name: 'user', type: ddb.AttributeType.STRING},
      sortKey: {name: 'job', type: ddb.AttributeType.STRING},
    })


    // Setup Project Bucket
    this.bucket = new s3.Bucket(this, 'TranscribeBucket', {
      versioned: true,
      removalPolicy: cdk.RemovalPolicy.DESTROY
    });

    // Setup ECS Cluster
    this.cluster = new ecs.Cluster(this, "TranscribeCluster", {
    });

    // Setup SQS Queue for Emails
    this.queue = new sqs.Queue(this, 'EmailQueue');
  
    // This is here due to S3 Event Notification needs to be setup here otherwise a circular dependency
    this.setupStartTranscriberLambda()
  }
  
  setupStartTranscriberLambda() {
    const startTranscribeFromS3Event = new lambda.GoFunction(this, 'StartTranscribeFromS3Event', {
      entry: '../cmd/StartTranscribeFromS3EventLambda/',
      environment: {
        TABLE_NAME: this.jobTable.tableName
      }
    })
    const s3PutEventSource = new lambdaEventSources.S3EventSource(this.bucket, {
      events: [
        s3.EventType.OBJECT_CREATED_PUT
      ],
      filters: [
        {
          prefix: 'user',
        }
      ]
    });
    startTranscribeFromS3Event.addEventSource(s3PutEventSource);
    this.jobTable.grantFullAccess(startTranscribeFromS3Event)
    this.bucket.grantReadWrite(startTranscribeFromS3Event)
    startTranscribeFromS3Event.role?.addManagedPolicy(
      ManagedPolicy.fromAwsManagedPolicyName('AmazonTranscribeFullAccess')
    );
  }
}
