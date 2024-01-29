#!/bin/bash

# Set variables for the ECR repository and image tag
ECR_REPOSITORY=$(aws ecr describe-repositories --repository-names plandex-ecr-repository | jq -r '.repositories[0].repositoryUri')
IMAGE_TAG=$(git rev-parse --short HEAD)

# Function to deploy or update the CloudFormation stack using AWS CDK
deploy_or_update_stack() {
  # Check if the stack exists
  STACK_NAME=$(aws cloudformation list-stacks --stack-status-filter CREATE_COMPLETE UPDATE_COMPLETE | jq -r '.StackSummaries[] | select(.StackName | startswith("plandex-stack-")) | .StackName')

  if [ -z "$STACK_NAME" ]; then
    # Deploy the stack if it does not exist
    npx cdk deploy "plandex-stack-*" --require-approval never --tags stack=$STACK_TAG
  else
    # Update the stack if it exists
    npx cdk deploy "$STACK_NAME" --require-approval never
  fi
}

# Function to build and push the Docker image to ECR
build_and_push_image() {
  # Login to ECR
  aws ecr get-login-password --region $(aws configure get region) | docker login --username AWS --password-stdin $ECR_REPOSITORY

  # Build the Docker image
  docker build -t plandex-server:$IMAGE_TAG -f app/Dockerfile.server .

  # Tag the image for the ECR repository
  docker tag plandex-server:$IMAGE_TAG $ECR_REPOSITORY:$IMAGE_TAG

  # Push the image to ECR
  docker push $ECR_REPOSITORY:$IMAGE_TAG
}

# Function to update the ECS service with the new Docker image
update_ecs_service() {
  # Extract the tag from the ECR repository URI
  TAG=$(echo $ECR_REPOSITORY | grep -oE 'plandex-ecr-repository-[a-zA-Z0-9]+' | sed 's/plandex-ecr-repository-//')

  # Use the extracted tag to find the ECS cluster and service names
  CLUSTER_NAME=$(aws ecs list-clusters | jq -r --arg TAG "$TAG" '.clusterArns[] | select(contains("plandex-ecs-cluster-" + $TAG)) | split("/")[1]')
  SERVICE_NAME=$(aws ecs list-services --cluster "$CLUSTER_NAME" | jq -r --arg TAG "$TAG" '.serviceArns[] | select(contains("plandex-fargate-service-" + $TAG)) | split("/")[1]')

  # Replace placeholders in ecs-container-definitions.json with actual values
  sed -i "s|\${ECR_REPOSITORY_URI}|$ECR_REPOSITORY|g" ecs-container-definitions.json
  sed -i "s|\${IMAGE_TAG}|$IMAGE_TAG|g" ecs-container-definitions.json
  sed -i "s|\${AWS_REGION}|$(aws configure get region)|g" ecs-container-definitions.json

  # Register a new task definition with the new image
  TASK_DEF_ARN=$(aws ecs register-task-definition --family "plandex-task-definition-$TAG" --container-definitions file://ecs-container-definitions.json | jq -r '.taskDefinition.taskDefinitionArn')

  # Update the ECS service to use the new task definition
  aws ecs update-service --cluster "$CLUSTER_NAME" --service "$SERVICE_NAME" --task-definition $TASK_DEF_ARN
}

# Deploy or update the CloudFormation stack
deploy_or_update_stack

# Build and push the Docker image to ECR
build_and_push_image

# Update the ECS service with the new Docker image
update_ecs_service
