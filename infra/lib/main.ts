import * as cdk from "@aws-cdk/core";

class PlandexApi extends cdk.Stack {
  constructor(scope: cdk.Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);
  }
}

const app = new cdk.App();
new PlandexApi(app, "plandex-api");
