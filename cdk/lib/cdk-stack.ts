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

    const bucket = new s3.Bucket(this, 'TranscribeBucket', {
      versioned: true,
      removalPolicy: cdk.RemovalPolicy.DESTROY
    });
    


    const fargateServer = new ecsPatterns.ApplicationLoadBalancedFargateService(this, "TranscribeFrontend", {
      cluster: cluster, // Required
      cpu: 512,
      desiredCount: 1,
      
      taskImageOptions: {
        containerPort: 8080,
        environment: {
          TABLE_NAME: table.tableName,
          PROJECT_BUCKET: bucket.bucketName
        },
        image: ecs.ContainerImage.fromAsset('..', {
          file: 'docker/frontend/Dockerfile',
          exclude: [ 'build', 'cdk' ],
        }),
      },
  
      memoryLimitMiB: 2048,
      publicLoadBalancer: true,
      
    });

    fargateServer.targetGroup.configureHealthCheck({path: "/ping"})

    bucket.grantReadWrite(fargateServer.taskDefinition.taskRole)
    table.grantFullAccess(fargateServer.taskDefinition.taskRole)

    

    const sendingEmail = "sean.c.nash@gmail.com"

    const sendEmailLambda = new lambda.GoFunction(this, 'SendEmailLambda', {
      entry: '../cmd/SendEmailLambda/',
      environment: {
        SENDING_EMAIL: sendingEmail,
        USER_POOL: userPool.userPoolId
      }
    })
    sendEmailLambda.addToRolePolicy(new PolicyStatement({
      actions: ['ses:SendEmail', 'SES:SendRawEmail'],
      resources: ['*'],
      effect: Effect.ALLOW,
    }));
    sendEmailLambda.addToRolePolicy(new PolicyStatement({
      actions: ['cognito-idp:AdminGetUser'],
      resources: [userPool.userPoolArn],
      effect: Effect.ALLOW,
    }));
    
  
    sendEmailLambda.addEventSource(new lambdaEventSources.SqsEventSource(queue, {
      batchSize: 10, // default
      maxBatchingWindow: cdk.Duration.minutes(1),
    }));

    const startTranscribeFromS3Event = new lambda.GoFunction(this, 'StartTranscribeFromS3Event', {
      entry: '../cmd/StartTranscribeFromS3EventLambda/',
      environment: {
        TABLE_NAME: table.tableName
      }
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
    table.grantFullAccess(startTranscribeFromS3Event)
    bucket.grantReadWrite(startTranscribeFromS3Event)
    startTranscribeFromS3Event.role?.addManagedPolicy(ManagedPolicy.fromAwsManagedPolicyName('AmazonTranscribeFullAccess'))

    const transcriberFinish = new lambda.GoFunction(this, 'TranscriberFinish', {
      entry: '../cmd/TranscriberFinishLambda/',
      environment: {
        'EMAIL_USER': sendingEmail,
        'USER_POOL': userPool.userPoolArn,
        'TABLE_NAME': table.tableName,
        'EMAIL_QUEUE_URL': queue.queueUrl
      },
    })
    table.grantReadWriteData(transcriberFinish)
    queue.grantSendMessages(transcriberFinish)

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
