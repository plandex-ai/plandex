import * as sns from "@aws-cdk/aws-sns";
import * as subscriptions from "@aws-cdk/aws-sns-subscriptions";
import * as cdk from "@aws-cdk/core";
import * as ec2 from "@aws-cdk/aws-ec2";
import * as rds from "@aws-cdk/aws-rds";
import * as secretsmanager from "@aws-cdk/aws-secretsmanager";
import * as ecs from "@aws-cdk/aws-ecs";
import * as ecr from "@aws-cdk/aws-ecr";
import * as efs from "@aws-cdk/aws-efs";
import * as elbv2 from "@aws-cdk/aws-elasticloadbalancingv2";
import * as iam from "@aws-cdk/aws-iam";
import * as acm from "@aws-cdk/aws-certificatemanager";
import * as cloudwatch from "@aws-cdk/aws-cloudwatch";
import * as cloudwatchActions from "@aws-cdk/aws-cloudwatch-actions";

const tag = process.env.STACK_TAG;
const notifyEmail = process.env.NOTIFY_EMAIL;
const notifySms = process.env.NOTIFY_SMS;
const certificateArn = process.env.CERTIFICATE_ARN;
const imageTag = process.env.IMAGE_TAG;

export class PlandexStack extends cdk.Stack {
  constructor(scope: cdk.Construct, id: string, props?: cdk.StackProps) {
    if (!tag) {
      throw new Error("STACK_TAG environment variable is not set");
    }
    if (!notifyEmail) {
      throw new Error("NOTIFY_EMAIL environment variable is not set");
    }
    if (!notifySms) {
      throw new Error("NOTIFY_SMS environment variable is not set");
    }
    if (!certificateArn) {
      throw new Error("CERTIFICATE_ARN environment variable is not set");
    }
    if (!imageTag) {
      throw new Error("IMAGE_TAG environment variable is not set");
    }

    super(scope, id, props);

    // Create a VPC with two public and two private subnets
    console.log("Creating VPC...");
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

    const rdsSecurityGroup = new ec2.SecurityGroup(this, "RdsSecurityGroup", {
      vpc,
      description: "Security group for RDS instance",
    });

    const efsSecurityGroup = new ec2.SecurityGroup(
      this,
      `plandex-efs-sg-${tag}`,
      {
        vpc,
      }
    );

    const fargateServiceSecurityGroup = new ec2.SecurityGroup(
      this,
      `plandex-fargate-service-sg-${tag}`,
      {
        vpc,
        allowAllOutbound: true, // Allows the containers to access the internet
      }
    );

    const albSecurityGroup = new ec2.SecurityGroup(this, "AlbSecurityGroup", {
      vpc,
      description: "Security group for the ALB",
    });

    // RDS security group rules
    fargateServiceSecurityGroup.addEgressRule(
      rdsSecurityGroup,
      ec2.Port.tcp(5432),
      "Allow outbound traffic from Fargate service to RDS instance"
    );

    rdsSecurityGroup.addIngressRule(
      fargateServiceSecurityGroup,
      ec2.Port.tcp(5432),
      "Allow Fargate service to access RDS instance"
    );

    // EFS security group rules
    efsSecurityGroup.addIngressRule(
      fargateServiceSecurityGroup,
      ec2.Port.tcp(2049),
      "Allow inbound NFS traffic from Fargate service"
    );

    fargateServiceSecurityGroup.addEgressRule(
      ec2.Peer.ipv4(vpc.vpcCidrBlock),
      ec2.Port.tcp(2049),
      "Allow outbound NFS traffic to EFS"
    );

    // ALB security group rules
    // Allow inbound HTTP and HTTPS traffic
    albSecurityGroup.addIngressRule(
      ec2.Peer.anyIpv4(),
      ec2.Port.tcp(80),
      "Allow inbound HTTP traffic"
    );
    albSecurityGroup.addIngressRule(
      ec2.Peer.anyIpv4(),
      ec2.Port.tcp(443),
      "Allow inbound HTTPS traffic"
    );
    // Allow traffic from the alb to the fargate service
    fargateServiceSecurityGroup.addIngressRule(
      albSecurityGroup,
      ec2.Port.tcp(8080),
      "Allow inbound traffic from ALB"
    );

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
    console.log("Creating RDS instance...");
    const dbInstance = new rds.DatabaseInstance(
      this,
      `plandex-rds-instance-${tag}`,
      {
        engine: rds.DatabaseInstanceEngine.postgres({
          version: rds.PostgresEngineVersion.VER_14,
        }),
        storageEncrypted: true,
        instanceType: ec2.InstanceType.of(
          ec2.InstanceClass.BURSTABLE4_GRAVITON,
          ec2.InstanceSize.MICRO
        ),
        vpc,
        vpcSubnets: {
          subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
        },
        securityGroups: [rdsSecurityGroup],
        credentials: rds.Credentials.fromSecret(dbCredentialsSecret), // Use credentials from Secrets Manager
        databaseName: "plandex",
      }
    );

    // Get the ECR repository
    const ecrRepository = ecr.Repository.fromRepositoryName(
      this,
      "PlandexEcrRepository",
      "plandex-ecr-repository"
    );

    // Create an ECS cluster
    console.log("Creating ECS cluster...");
    const ecsCluster = new ecs.Cluster(this, `plandex-ecs-cluster-${tag}`, {
      vpc,
    });

    // Create an EFS file system
    const fileSystem = new efs.FileSystem(
      this,
      `plandex-efs-file-system-${tag}`,
      {
        vpc,
        securityGroup: efsSecurityGroup,
        vpcSubnets: {
          subnetType: ec2.SubnetType.PRIVATE_ISOLATED, // Deploy the file system in private subnets
        },
        encrypted: true,
        enableAutomaticBackups: true,
      }
    );

    // Create an IAM role for the Fargate task to interact with SES
    const taskRole = new iam.Role(this, `plandex-task-role-${tag}`, {
      assumedBy: new iam.ServicePrincipal("ecs-tasks.amazonaws.com"),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName("AmazonSESFullAccess"), // Ensure this policy allows sending emails
      ],
    });

    const taskExecutionRole = new iam.Role(this, "TaskExecutionRole", {
      assumedBy: new iam.ServicePrincipal("ecs-tasks.amazonaws.com"),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName(
          "service-role/AmazonECSTaskExecutionRolePolicy"
        ),
        iam.ManagedPolicy.fromAwsManagedPolicyName(
          "AmazonElasticFileSystemClientReadWriteAccess"
        ),
      ],
    });
    dbCredentialsSecret.grantRead(taskExecutionRole);

    // Create a Fargate task definition with EFS volume
    const taskDefinition = new ecs.FargateTaskDefinition(
      this,
      `plandex-task-definition-${tag}`,
      {
        memoryLimitMiB: 512,
        cpu: 256,
        taskRole,
        executionRole: taskExecutionRole,
      }
    );

    // Add a container to the task definition
    const container = taskDefinition.addContainer("plandex-server", {
      healthCheck: {
        command: [
          "CMD-SHELL",
          "curl -f http://localhost:8080/health || exit 1"
        ],
        interval: cdk.Duration.seconds(30),
        timeout: cdk.Duration.seconds(5),
        retries: 3,
        startPeriod: cdk.Duration.seconds(60),
      },
      image: ecs.ContainerImage.fromEcrRepository(ecrRepository, imageTag),
      logging: new ecs.AwsLogDriver({ streamPrefix: "plandex-server" }),
      environment: {
        // give time for streams to finish before container is stopped
        ECS_CONTAINER_STOP_TIMEOUT: "60m",
        DB_HOST: dbInstance.dbInstanceEndpointAddress,
        DB_PORT: dbInstance.dbInstanceEndpointPort,
        DB_NAME: "plandex",
        DB_USER: "dbadmin",
        GOENV: "production",
        IS_CLOUD: "1",
      },
      secrets: {
        DB_PASSWORD: ecs.Secret.fromSecretsManager(
          dbCredentialsSecret,
          "password"
        ),
      },
      portMappings: [
        {
          containerPort: 8080,
          hostPort: 8080,
          protocol: ecs.Protocol.TCP,
        },
      ],
    });

    // Mount the EFS file system to the container
    const volumeName = "plandex-efs-volume";
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

    const fargateService = new ecs.FargateService(
      this,
      `plandex-fargate-service-${tag}`,
      {
        cluster: ecsCluster,
        taskDefinition,
        desiredCount: 1,
        securityGroups: [fargateServiceSecurityGroup],
        vpcSubnets: {
          subnetType: ec2.SubnetType.PUBLIC, // Deploy tasks in public subnets
        },
        assignPublicIp: true, // Assign public IP addresses to tasks for outbound internet access
      }
    );

    // Define the scaling target
    const scaling = fargateService.autoScaleTaskCount({
      minCapacity: 1,
      maxCapacity: 10,
    });

    // Define the CPU-based scaling policy
    scaling.scaleOnCpuUtilization("CpuScaling", {
      targetUtilizationPercent: 50,
      scaleInCooldown: cdk.Duration.seconds(300),
      scaleOutCooldown: cdk.Duration.seconds(300),
    });

    // Define the memory-based scaling policy
    scaling.scaleOnMemoryUtilization("MemoryScaling", {
      targetUtilizationPercent: 50,
      scaleInCooldown: cdk.Duration.seconds(300),
      scaleOutCooldown: cdk.Duration.seconds(300),
    });
    this,
      `plandex-fargate-service-${tag}`,
      {
        cluster: ecsCluster,
        taskDefinition,
        desiredCount: 1,
        securityGroups: [fargateServiceSecurityGroup],
      };

    // Create an Application Load Balancer in the VPC
    const alb = new elbv2.ApplicationLoadBalancer(this, `plandex-alb-${tag}`, {
      vpc,
      internetFacing: true,
      securityGroup: albSecurityGroup,
      vpcSubnets: {
        subnetType: ec2.SubnetType.PUBLIC,
      },
    });

    // Create an HTTP listener and add a redirect to HTTPS
    const httpListener = alb.addListener("HttpListener", {
      port: 80,
      open: true,
    });
    httpListener.addRedirectResponse("HTTPtoHTTPSRedirect", {
      statusCode: "HTTP_301",
      protocol: "HTTPS",
      port: "443",
      host: "#{host}",
      path: "/#{path}",
      query: "#{query}",
    });

    // Create an HTTPS listener
    const httpsListener = alb.addListener("HttpsListener", {
      port: 443,
      certificates: [
        acm.Certificate.fromCertificateArn(this, "Certificate", certificateArn),
      ],
    });

    // Add a target group for the ECS service to the HTTPS listener
    const targetGroup = httpsListener.addTargets("plandexEcsTarget", {
      healthCheck: {
        path: "/health",
        port: "8080",
        protocol: elbv2.Protocol.HTTP,
        healthyHttpCodes: "200",
        interval: cdk.Duration.seconds(30),
        timeout: cdk.Duration.seconds(5),
        healthyThresholdCount: 2,
        unhealthyThresholdCount: 3,
      },
      port: 8080,
      targets: [fargateService],
    });

    // Create CloudWatch alarms for high CPU and memory utilization
    const alarmNotificationTopic = new sns.Topic(
      this,
      "AlarmNotificationTopic",
      {
        displayName: "Alarm Notifications",
      }
    );

    alarmNotificationTopic.addSubscription(
      new subscriptions.EmailSubscription(notifyEmail)
    );
    alarmNotificationTopic.addSubscription(
      new subscriptions.SmsSubscription(notifySms)
    );

    const dbInstanceCpuUtilizationAlarm = new cloudwatch.Alarm(
      this,
      "HighDbCpuUtilizationAlarm",
      {
        metric: dbInstance.metricCPUUtilization(),
        threshold: 80,
        evaluationPeriods: 2,
        alarmDescription: "Alarm when RDS CPU utilization exceeds 80%",
        comparisonOperator:
          cloudwatch.ComparisonOperator.GREATER_THAN_THRESHOLD,
        treatMissingData: cloudwatch.TreatMissingData.NOT_BREACHING,
      }
    );
    dbInstanceCpuUtilizationAlarm.addAlarmAction(
      new cloudwatchActions.SnsAction(alarmNotificationTopic)
    );

    const dbInstanceFreeableMemoryAlarm = new cloudwatch.Alarm(
      this,
      "LowDbFreeableMemoryAlarm",
      {
        metric: dbInstance.metricFreeableMemory(),
        threshold: 200 * 1024 * 1024, // Threshold in bytes (e.g., 200MB)
        evaluationPeriods: 2,
        alarmDescription: "Alarm when RDS freeable memory is too low",
        comparisonOperator: cloudwatch.ComparisonOperator.LESS_THAN_THRESHOLD,
        treatMissingData: cloudwatch.TreatMissingData.NOT_BREACHING,
      }
    );
    dbInstanceFreeableMemoryAlarm.addAlarmAction(
      new cloudwatchActions.SnsAction(alarmNotificationTopic)
    );

    const fargateServiceCpuUtilizationAlarm = new cloudwatch.Alarm(
      this,
      "HighFargateCpuUtilizationAlarm",
      {
        metric: fargateService.metricCpuUtilization(),
        threshold: 80,
        evaluationPeriods: 2,
        alarmDescription: "Alarm when Fargate CPU utilization exceeds 80%",
        comparisonOperator:
          cloudwatch.ComparisonOperator.GREATER_THAN_THRESHOLD,
        treatMissingData: cloudwatch.TreatMissingData.NOT_BREACHING,
      }
    );
    fargateServiceCpuUtilizationAlarm.addAlarmAction(
      new cloudwatchActions.SnsAction(alarmNotificationTopic)
    );

    const fargateServiceMemoryUtilizationAlarm = new cloudwatch.Alarm(
      this,
      "HighFargateMemoryUtilizationAlarm",
      {
        metric: fargateService.metricMemoryUtilization(),
        threshold: 80,
        evaluationPeriods: 2,
        alarmDescription: "Alarm when Fargate memory utilization exceeds 80%",
        comparisonOperator:
          cloudwatch.ComparisonOperator.GREATER_THAN_THRESHOLD,
        treatMissingData: cloudwatch.TreatMissingData.NOT_BREACHING,
      }
    );
    fargateServiceMemoryUtilizationAlarm.addAlarmAction(
      new cloudwatchActions.SnsAction(alarmNotificationTopic)
    );
  }
}

const app = new cdk.App();

new PlandexStack(app, "plandex-stack-" + tag);
