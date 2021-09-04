#!/usr/bin/env node
import * as cdk from '@aws-cdk/core';
import { TranscriberStack } from '../lib/cdk-stack';

const app = new cdk.App();
new TranscriberStack(app, 'TranscriberStack');
