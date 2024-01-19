import * as cdk from '@aws-cdk/core';

class MyCdkStack extends cdk.Stack {
  constructor(scope: cdk.Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // Define your resources here
  }
}

const app = new cdk.App();
new MyCdkStack(app, 'MyCdkStack');
