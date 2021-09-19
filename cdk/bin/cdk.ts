#!/usr/bin/env node
import * as cdk from '@aws-cdk/core';
import { FrontendStack } from '../lib/frontend-stack';
import { InfraStack} from '../lib/infra-stack'
import { LambdaStack } from '../lib/lambda-stack'
const app = new cdk.App();

const infraStack = new InfraStack(app, 'InfraStack')

const frontendStack = new FrontendStack(app, 'FrontendStack', {
    bucket: infraStack.bucket,
    cluster: infraStack.cluster,
    jobTable: infraStack.jobTable,
    queue: infraStack.queue,
    userPool: infraStack.userPool
});

const lambdaStack = new LambdaStack(app, 'LambdaStack', {
    bucket: infraStack.bucket,
    cluster: infraStack.cluster,
    jobTable: infraStack.jobTable,
    queue: infraStack.queue,
    userPool: infraStack.userPool
});