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

export class TranscriberStack extends cdk.Stack {
  constructor(scope: cdk.App, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    const userPool = new cognito.UserPool(this, 'UserPool', {
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
      userPool: userPool,
      generateSecret: false,
      authFlows: {
        userPassword: true
      }
    }) 

    const userPoolGroup = new cognito.CfnUserPoolGroup(this, 'TranscriberAPI', {
      userPoolId: userPool.userPoolId,
      groupName: 'jo',
      
    })
    
    const table = new ddb.Table(this, 'Job', {
      billingMode: ddb.BillingMode.PROVISIONED,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
      partitionKey: {name: 'job', type: ddb.AttributeType.STRING}, 
    })

    table.addGlobalSecondaryIndex({
      indexName: 'user-index',
      partitionKey:  {name: 'user', type: ddb.AttributeType.STRING},
      sortKey: {name: 'job', type: ddb.AttributeType.STRING},
    })
    const vpc = new ec2.Vpc(this, 'VPC', {
      maxAzs: 3 // Default is all AZs in region
    });

    const bucket = new s3.Bucket(this, 'MyFirstBucket', {
      versioned: true
    });
    
    const cluster = new ecs.Cluster(this, "MyCluster", {
    //  vpc: vpc
    });
    

    const fargateServer = new ecsPatterns.ApplicationLoadBalancedFargateService(this, "MyFargateService", {
      cluster: cluster, // Required
      cpu: 512, // Default is 256
      desiredCount: 1, // Default is 1
      taskImageOptions: {
        environment: {
          TABLE_NAME: table.tableArn,
          PROJECT_BUCKET: bucket.bucketArn
        },
        image: ecs.ContainerImage.fromAsset('..', {
          file: 'docker/frontend/Dockerfile',
          exclude: [ 'build', 'cdk' ],
        })
      },
      memoryLimitMiB: 2048, // Default is 512
      publicLoadBalancer: true // Default is false
    });

      
    const queue = new sqs.Queue(this, 'EmailQueue');



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

    const transcriberFinish = new lambda.GoFunction(this, 'TranscriberFinish', {
      entry: '../cmd/TranscriberFinishLambda/',
      environment: {
        'EMAIL_USER': sendingEmail,
        'USER_POOL': userPool.userPoolArn
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
