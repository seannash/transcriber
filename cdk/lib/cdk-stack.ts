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

export class TranscriberStack extends cdk.Stack {
  constructor(scope: cdk.App, id: string, props?: cdk.StackProps) {
    super(scope, id, props);
    
    const vpc = new ec2.Vpc(this, 'VPC', {
      maxAzs: 3 // Default is all AZs in region
    });

    const cluster = new ecs.Cluster(this, "MyCluster", {
      vpc: vpc
    });

    // Create a load-balanced Fargate service and make it public
    /*
    new ecsPatterns.ApplicationLoadBalancedFargateService(this, "MyFargateService", {
      cluster: cluster, // Required
      cpu: 512, // Default is 256
      desiredCount: 6, // Default is 1
      taskImageOptions: {
        image: ecs.ContainerImage.fromAsset('..', {
          file: 'docker/frontend/Dockerfile',
          exclude: [ 'build', 'cdk' ]
        })
      },
      memoryLimitMiB: 2048, // Default is 512
      publicLoadBalancer: true // Default is false
    });
*/

/*
 TranscribeTable:
    Type: "AWS::DynamoDB::Table"
    Properties:
      AttributeDefinitions: 
        - 
          AttributeName: "job"
          AttributeType: "S"
        - 
          AttributeName: "user"
          AttributeType: "S"
      KeySchema: 
        -
          AttributeName: "job"
          KeyType: "HASH"
      ProvisionedThroughput:
        ReadCapacityUnits: 1
        WriteCapacityUnits: 1
      GlobalSecondaryIndexes:
        -
          IndexName: "user-index"
          KeySchema:
            -
              AttributeName: "user"
              KeyType: "HASH"
          Projection:
            ProjectionType: "ALL"
          ProvisionedThroughput:
            ReadCapacityUnits: 1
            WriteCapacityUnits: 1
            */

    const table = new ddb.Table(this, 'Job', {
      billingMode: ddb.BillingMode.PROVISIONED,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
      partitionKey: {name: 'job', type: ddb.AttributeType.STRING},
      //sortKey: {name: 'createdAt', type: ddb.AttributeType.NUMBER},      
    })

    table.addGlobalSecondaryIndex({
      indexName: 'user-index',
      partitionKey:  {name: 'user', type: ddb.AttributeType.STRING},
      sortKey: {name: 'job', type: ddb.AttributeType.STRING},
    })
      
    const queue = new sqs.Queue(this, 'EmailQueue');

    const bucket = new s3.Bucket(this, 'MyFirstBucket', {
      versioned: true
    });

    const sendEmailLambda = new lambda.GoFunction(this, 'SendEmailLambda', {
      entry: '../cmd/SendEmailLambda/',
    })
    sendEmailLambda.addEventSource(new lambdaEventSources.SqsEventSource(queue, {
      batchSize: 10, // default
      maxBatchingWindow: cdk.Duration.minutes(1),
    }));

    const startTranscribeFromS3Event = new lambda.GoFunction(this, 'StartTranscribeFromS3Event', {
      entry: '../cmd/StartTranscribeFromS3EventLambda/',
    })
    const s3PutEventSource = new lambdaEventSources.S3EventSource(bucket, {
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

    const sendingEmail = "sean.c.nash@gmail.com"
    var userPool =""
    const transcriberFinish = new lambda.GoFunction(this, 'TranscriberFinish', {
      entry: '../cmd/TranscriberFinishLambda/',
      environment: {
        'EMAIL_USER': sendingEmail,
        'USER_POOL': userPool
      },
    })
    const finishRule = new events.Rule(this, 'FinishRule', {
      ruleName: 'FinishRule',
      eventPattern: {
        source: [ 'aws.transcribe' ],
        detailType: [ 'Transcribe Job State Change' ]
      },
      targets: [ new targets.LambdaFunction(transcriberFinish) ]
    })

  }

}
