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

export interface FrontendProps extends cdk.StackProps {
  readonly bucket: s3.Bucket
  readonly cluster: ecs.Cluster;
  readonly jobTable: ddb.Table;
  readonly queue: sqs.Queue;
  readonly userPool: cognito.UserPool;
}

export class FrontendStack extends cdk.Stack {
  constructor(scope: cdk.App, id: string, props: FrontendProps) {
    super(scope, id, props);

    const fargateServer = new ecsPatterns.ApplicationLoadBalancedFargateService(this, "TranscribeFrontend", {
      cluster: props.cluster,
      cpu: 512,
      desiredCount: 1,
      
      taskImageOptions: {
        containerPort: 8080,
        environment: {
          TABLE_NAME: props.jobTable.tableName,
          PROJECT_BUCKET: props.bucket.bucketName
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
    props.bucket.grantReadWrite(fargateServer.taskDefinition.taskRole)
    props.jobTable.grantFullAccess(fargateServer.taskDefinition.taskRole)

  
  }

}
