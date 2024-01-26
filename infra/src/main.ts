import * as cdk from "@aws-cdk/core";
import * as ec2 from "@aws-cdk/aws-ec2";
import * as rds from "@aws-cdk/aws-rds";
import * as secretsmanager from "@aws-cdk/aws-secretsmanager";
import * as ecs from "@aws-cdk/aws-ecs";
import * as ecr from "@aws-cdk/aws-ecr";
import * as efs from "@aws-cdk/aws-efs";
import * as iam from "@aws-cdk/aws-iam";
import * as ses from "@aws-cdk/aws-ses";
import { v4 as uuid } from "uuid";

const tag = uuid().split("-")[0];

export class MyStack extends cdk.Stack {
  constructor(scope: cdk.Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // Create a VPC with two public and two private subnets
    const vpc = new ec2.Vpc(this, `plandex-vpc-${tag}`, {
      maxAzs: 2,
      subnetConfiguration: [
        {
          cidrMask: 24,
          name: "PublicSubnet",
          subnetType: ec2.SubnetType.PUBLIC,
        },
        {
          cidrMask: 24,
          name: "PrivateSubnet",
          subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
        },
      ],
    });

    // Create a Secrets Manager secret to store the RDS database credentials
    const dbCredentialsSecret = new secretsmanager.Secret(
      this,
      `plandex-db-credentials-secret-${tag}`,
      {
        generateSecretString: {
          secretStringTemplate: JSON.stringify({ username: "dbadmin" }),
          generateStringKey: "password",
          excludeCharacters: '"@/\\',
        },
      }
    );

    // Create an RDS Aurora PostgreSQL database
    const dbInstance = new rds.DatabaseInstance(
      this,
      `plandex-rds-instance-${tag}`,
      {
        engine: rds.DatabaseInstanceEngine.postgres({
          version: rds.PostgresEngineVersion.VER_14_2,
        }),
        instanceType: ec2.InstanceType.of(
          ec2.InstanceClass.BURSTABLE2,
          ec2.InstanceSize.MICRO
        ),
        vpc,
        vpcSubnets: {
          subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
        },
        credentials: rds.Credentials.fromSecret(dbCredentialsSecret), // Use credentials from Secrets Manager
      }
    );

    // Create an ECR repository
    const ecrRepository = new ecr.Repository(
      this,
      `plandex-ecr-repository-${tag}`
    );

    // Create an ECS cluster
    const ecsCluster = new ecs.Cluster(this, `plandex-ecs-cluster-${tag}`, {
      vpc,
    });

    // Create an EFS file system
    const fileSystem = new efs.FileSystem(
      this,
      `plandex-efs-file-system-${tag}`,
      {
        vpc,
      }
    );

    // Create an IAM role for the Fargate task to interact with SES
    const taskRole = new iam.Role(this, `plandex-task-role-${tag}`, {
      assumedBy: new iam.ServicePrincipal("ecs-tasks.amazonaws.com"),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName("AmazonSESFullAccess"),
      ],
    });

    // Create a Fargate task definition with EFS volume
    const taskDefinition = new ecs.FargateTaskDefinition(
      this,
      `plandex-task-definition-${tag}`,
      {
        memoryLimitMiB: 512,
        cpu: 256,
        taskRole,
      }
    );

    // Add a container to the task definition
    const container = taskDefinition.addContainer("MyContainer", {
      image: ecs.ContainerImage.fromEcrRepository(ecrRepository),
      logging: new ecs.AwsLogDriver({ streamPrefix: "MyContainer" }),
      environment: {
        ECS_CONTAINER_STOP_TIMEOUT: "10m", // gives time for streams to finish before container is stopped
      },
    });

    // Mount the EFS file system to the container
    const volumeName = "MyEfsVolume";
    taskDefinition.addVolume({
      name: volumeName,
      efsVolumeConfiguration: {
        fileSystemId: fileSystem.fileSystemId,
      },
    });
    container.addMountPoints({
      sourceVolume: volumeName,
      containerPath: "/plandex-server",
      readOnly: false,
    });

    // Create a Fargate service with a security group that allows outbound internet access and access to the RDS database
    const fargateServiceSecurityGroup = new ec2.SecurityGroup(
      this,
      `plandex-fargate-service-sg-${tag}`,
      {
        vpc,
        allowAllOutbound: true, // Allows the containers to access the internet
      }
    );

    // Define the ingress rule for the security group to allow the Fargate service to communicate with the RDS instance
    fargateServiceSecurityGroup.addIngressRule(
      ec2.Peer.anyIpv4(),
      ec2.Port.tcp(parseInt(dbInstance.dbInstanceEndpointPort)),
      "Allow Fargate service to access RDS instance"
    );

    const fargateService = new ecs.FargateService(
      this,
      `plandex-fargate-service-${tag}`,
      {
        cluster: ecsCluster,
        taskDefinition,
        desiredCount: 1,
        securityGroups: [fargateServiceSecurityGroup],
      }
    );

    // Create an SES email sending resource
    // Note: SES setup can vary based on the region and verification status. Here, we're creating a simple rule set.
    const emailSendingResource = new ses.CfnReceiptRuleSet(
      this,
      `plandex-email-sending-resource-${tag}`,
      {
        ruleSetName: "MyRuleSet",
      }
    );
  }
}

const app = new cdk.App();
new MyStack(app, "plandex-stack-" + tag);
