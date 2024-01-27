#!/bin/bash

# Set variables for the ECR repository and image tag
ECR_REPOSITORY=$(aws ecr describe-repositories --repository-names plandex-ecr-repository | jq -r '.repositories[0].repositoryUri')
IMAGE_TAG=$(git rev-parse --short HEAD)

# Function to deploy or update the CloudFormation stack using AWS CDK
deploy_or_update_stack() {
  # Check if the stack exists
  STACK_EXISTS=$(aws cloudformation describe-stacks --stack-name plandex-stack | jq -r '.Stacks | length')

  if [ "$STACK_EXISTS" -eq "0" ]; then
    # Deploy the stack if it does not exist
    npx cdk deploy "plandex-stack-*" --require-approval never
  else
    # Update the stack if it exists
    npx cdk deploy "plandex-stack-*" --require-approval never
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
  # Replace placeholders in ecs-container-definitions.json with actual values
  sed -i "s|\${ECR_REPOSITORY_URI}|$ECR_REPOSITORY|g" ecs-container-definitions.json
  sed -i "s|\${IMAGE_TAG}|$IMAGE_TAG|g" ecs-container-definitions.json
  sed -i "s|\${AWS_REGION}|$(aws configure get region)|g" ecs-container-definitions.json
  # Register a new task definition with the new image
  TASK_DEF_ARN=$(aws ecs register-task-definition --family plandex-task-definition --container-definitions file://ecs-container-definitions.json | jq -r '.taskDefinition.taskDefinitionArn')

  # Update the ECS service to use the new task definition
  aws ecs update-service --cluster plandex-ecs-cluster --service plandex-fargate-service --task-definition $TASK_DEF_ARN
}

# Deploy or update the CloudFormation stack
deploy_or_update_stack

# Build and push the Docker image to ECR
build_and_push_image

# Update the ECS service with the new Docker image
update_ecs_service
