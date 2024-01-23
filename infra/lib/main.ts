import * as cdk from '@aws-cdk/core';
import * as ec2 from '@aws-cdk/aws-ec2';
import * as rds from '@aws-cdk/aws-rds';
import * as secretsmanager from '@aws-cdk/aws-secretsmanager';
import * as ecs from '@aws-cdk/aws-ecs';
import * as ecr from '@aws-cdk/aws-ecr';
import * as efs from '@aws-cdk/aws-efs';
import * as iam from '@aws-cdk/aws-iam';
import * as ses from '@aws-cdk/aws-ses';

export class MyStack extends cdk.Stack {
  constructor(scope: cdk.Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // Create a VPC with two public and two private subnets
    const vpc = new ec2.Vpc(this, 'MyVpc', {
      maxAzs: 2,
      subnetConfiguration: [
        {
          cidrMask: 24,
          name: 'PublicSubnet',
          subnetType: ec2.SubnetType.PUBLIC,
        },
        {
          cidrMask: 24,
          name: 'PrivateSubnet',
          subnetType: ec2.SubnetType.PRIVATE_WITH_NAT,
        },
      ],
    });

    // Create a Secrets Manager secret to store the RDS database credentials
    const dbCredentialsSecret = new secretsmanager.Secret(this, 'DBCredentialsSecret', {
      generateSecretString: {
        secretStringTemplate: JSON.stringify({ username: 'dbadmin' }),
        generateStringKey: 'password',
        excludeCharacters: '"@/\\',
      },
    });

    // Create an RDS Aurora PostgreSQL database
    const dbInstance = new rds.DatabaseInstance(this, 'MyRdsInstance', {
      engine: rds.DatabaseInstanceEngine.postgres({
        version: rds.PostgresEngineVersion.VER_12_4,
      }),
      instanceType: ec2.InstanceType.of(ec2.InstanceClass.BURSTABLE2, ec2.InstanceSize.MICRO),
      vpc,
      vpcSubnets: {
        subnetType: ec2.SubnetType.PRIVATE_WITH_NAT,
      },
      credentials: rds.Credentials.fromSecret(dbCredentialsSecret), // Use credentials from Secrets Manager
      multiAz: false,
      allocatedStorage: 20,
      maxAllocatedStorage: 100,
    });

    // Create an ECR repository
    const ecrRepository = new ecr.Repository(this, 'MyEcrRepository');

    // Create an ECS cluster
    const ecsCluster = new ecs.Cluster(this, 'MyEcsCluster', {
      vpc,
    });

    // Create an EFS file system
    const fileSystem = new efs.FileSystem(this, 'MyEfsFileSystem', {
      vpc,
      lifecyclePolicy: efs.LifecyclePolicy.AFTER_14_DAYS, // Automatically delete files not accessed for 14 days
      performanceMode: efs.PerformanceMode.GENERAL_PURPOSE,
      throughputMode: efs.ThroughputMode.BURSTING,
    });

    // Create a Fargate task definition with EFS volume
    const taskDefinition = new ecs.FargateTaskDefinition(this, 'MyTaskDefinition', {
      memoryLimitMiB: 512,
      cpu: 256,
    });

    // Add a container to the task definition
    const container = taskDefinition.addContainer('MyContainer', {
      image: ecs.ContainerImage.fromEcrRepository(ecrRepository),
      logging: new ecs.AwsLogDriver({ streamPrefix: 'MyContainer' }),
    });

    // Mount the EFS file system to the container
    const volumeName = 'MyEfsVolume';
    taskDefinition.addVolume({
      name: volumeName,
      efsVolumeConfiguration: {
        fileSystemId: fileSystem.fileSystemId,
      },
    });
    container.addMountPoints({
      sourceVolume: volumeName,
      containerPath: '/mnt/efs',
      readOnly: false,
    });

    // Create a Fargate service with a security group that allows outbound internet access and access to the RDS database
    const fargateServiceSecurityGroup = new ec2.SecurityGroup(this, 'FargateServiceSG', {
      vpc,
      allowAllOutbound: true, // Allows the containers to access the internet
    });

    // Define the ingress rule for the security group to allow the Fargate service to communicate with the RDS instance
    fargateServiceSecurityGroup.addIngressRule(
      ec2.Peer.anyIpv4(),
      ec2.Port.tcp(dbInstance.dbInstanceEndpointPort),
      'Allow Fargate service to access RDS instance'
    );

    const fargateService = new ecs.FargateService(this, 'MyFargateService', {
      cluster: ecsCluster,
      taskDefinition,
      desiredCount: 1,
      securityGroups: [fargateServiceSecurityGroup],
    });

    // Create an IAM role for the Fargate task to interact with SES
    const taskRole = new iam.Role(this, 'FargateTaskRole', {
      assumedBy: new iam.ServicePrincipal('ecs-tasks.amazonaws.com'),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonSESFullAccess'),
      ],
    });

    // Attach the IAM role to the task definition
    taskDefinition.taskRole = taskRole;

    // Create an SES email sending resource
    // Note: SES setup can vary based on the region and verification status. Here, we're creating a simple rule set.
    const emailSendingResource = new ses.CfnReceiptRuleSet(this, 'MyEmailSendingResource', {
      ruleSetName: 'MyRuleSet',
    });

    // The SES rule set and other configurations would be set up here.
    // ...

    // Grant the Fargate task permission to send emails via SES
    // This is achieved by the managed policy attached to the task role.

    // Create an IAM role for the Fargate task to interact with SES
    const taskRole = new iam.Role(this, 'FargateTaskRole', {
      assumedBy: new iam.ServicePrincipal('ecs-tasks.amazonaws.com'),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonSESFullAccess'),
      ],
    });

    // Attach the IAM role to the task definition
    taskDefinition.taskRole = taskRole;

    // Create an SES email sending resource
    // Note: SES setup can vary based on the region and verification status. Here, we're creating a simple rule set.
    const emailSendingResource = new ses.CfnReceiptRuleSet(this, 'MyEmailSendingResource', {
      ruleSetName: 'MyRuleSet',
    });

    // The SES rule set and other configurations would be set up here.
    // ...

    // Grant the Fargate task permission to send emails via SES
    // This is achieved by the managed policy attached to the task role.
      engine: rds.DatabaseInstanceEngine.postgres({
        version: rds.PostgresEngineVersion.VER_12_4,
      }),
      instanceType: ec2.InstanceType.of(ec2.InstanceClass.BURSTABLE2, ec2.InstanceSize.MICRO),
      vpc,
      vpcSubnets: {
        subnetType: ec2.SubnetType.PRIVATE_WITH_NAT,
      },
      credentials: rds.Credentials.fromSecret(dbCredentialsSecret), // Use credentials from Secrets Manager
      multiAz: false,
      allocatedStorage: 20,
      maxAllocatedStorage: 100,
    });
  }
}

const app = new cdk.App();
new MyStack(app, 'MyStack');
