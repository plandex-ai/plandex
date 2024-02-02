#!/bin/bash

# Enhanced error handling and logging
set -e
set -o pipefail

# Initialize flag variable
SKIP_DEPLOY=false

# Parse command-line arguments
while [[ "$#" -gt 0 ]]; do
  case $1 in
      --image-only) SKIP_DEPLOY=true ;;
      *) echo "Unknown parameter passed: $1"; exit 1 ;;
  esac
  shift
done

log() {
  echo "[$(date +%Y-%m-%dT%H:%M:%S%z)]: $*"
}

handle_error() {
  local exit_code=$?
  log "An error occurred. Exiting with status ${exit_code}"
  exit $exit_code
}

trap 'handle_error' ERR

log() {
  echo "[$(date +%Y-%m-%dT%H:%M:%S%z)]: $*"
}

handle_error() {
  local exit_code=$?
  log "An error occurred. Exiting with status ${exit_code}"
  exit $exit_code
}

trap 'handle_error' ERR

if [ "$SKIP_DEPLOY" = true ]; then
  log "Skipping infra deploy due to --image-only flag"
else
  log "Deploying the infrastructure..."
fi

export AWS_PROFILE=plandex

# Generate a unique tag for the deployment
# Path to the file where the STACK_TAG is stored
STACK_TAG_FILE="stack-tag.txt"

# Function to generate a new STACK_TAG and save it to the file
generate_and_save_stack_tag() {
  export STACK_TAG=$(uuidgen | cut -d '-' -f1)
  echo $STACK_TAG > $STACK_TAG_FILE
  log "Generated new STACK_TAG: $STACK_TAG"
}

# Function to load the existing STACK_TAG from the file
load_stack_tag() {
  export STACK_TAG=$(cat $STACK_TAG_FILE)
  log "Loaded existing STACK_TAG: $STACK_TAG"
}

# Check if the STACK_TAG file exists and load it, otherwise generate a new one
if [ -f "$STACK_TAG_FILE" ]; then
  log "Loading existing STACK_TAG from file..."
  load_stack_tag
else
  log "Generating new STACK_TAG and saving to file..."
  generate_and_save_stack_tag
fi

# Function to ensure the ECR repository exists
ensure_ecr_repository_exists() {
  # Check if the ECR repository exists
  log "Checking if the ECR repository 'plandex-ecr-repository' exists..."
  if ! aws ecr describe-repositories --repository-names plandex-ecr-repository 2>/dev/null; then
    log "ECR repository does not exist. Creating repository..."
    aws ecr create-repository --repository-name plandex-ecr-repository
    log "ECR repository 'plandex-ecr-repository' created."
  else
    log "ECR repository 'plandex-ecr-repository' already exists."
  fi
}

log "Checking if the ECR repository exists..."

# Ensure the ECR repository exists before proceeding
ensure_ecr_repository_exists

# Set variables for the ECR repository and image tag
ECR_REPOSITORY=$(aws ecr describe-repositories --repository-names plandex-ecr-repository | jq -r '.repositories[0].repositoryUri')
export IMAGE_TAG=$(git rev-parse --short HEAD)

log "IMAGE_TAG: $IMAGE_TAG"

IS_NEW_STACK=false

# Function to deploy or update the CloudFormation stack using AWS CDK
deploy_or_update_stack() {
  # Check if the stack exists
  STACK_NAME=$(aws cloudformation list-stacks --stack-status-filter CREATE_COMPLETE UPDATE_COMPLETE | jq -r '.StackSummaries[] | select(.StackName | startswith("plandex-stack-")) | .StackName')

  if [ -z "$STACK_NAME" ]; then
    # Deploy the stack if it does not exist
    npx cdk deploy --require-approval never --app "npx ts-node src/main.ts" --context stackTag=$STACK_TAG "plandex-stack-$STACK_TAG"
    IS_NEW_STACK=true
  else
    # Update the stack if it exists
    npx cdk deploy "$STACK_NAME" --require-approval never --app "npx ts-node src/main.ts"
  fi
}

# Function to build and push the Docker image to ECR
build_and_push_image() {
  # Login to ECR
  aws ecr get-login-password --region $(aws configure get region) | docker login --username AWS --password-stdin $ECR_REPOSITORY

  # Build the Docker image
  cd ../app
  docker build -t plandex-server:$IMAGE_TAG -f Dockerfile.server .
  cd -

  # Tag the image for the ECR repository
  docker tag plandex-server:$IMAGE_TAG $ECR_REPOSITORY:$IMAGE_TAG

  # Push the image to ECR
  docker push $ECR_REPOSITORY:$IMAGE_TAG
}

# Function to update the ECS service with the new Docker image
update_ecs_service() {
  log "STACK_TAG: $STACK_TAG"

  # Use the extracted tag to find the ECS cluster and service names
  CLUSTER_NAME=$(aws ecs list-clusters | jq -r --arg TAG "$STACK_TAG" '.clusterArns[] | select(contains("plandex-stack-" + $TAG)) | split("/")[1]')
  SERVICE_NAME=$(aws ecs list-services --cluster "$CLUSTER_NAME" | jq -r --arg TAG "$STACK_TAG" '.serviceArns[] | select(contains("plandex-stack-" + $TAG)) | split("/")[1]')

  TASK_DEF_NAME=$(aws ecs list-task-definitions | jq -r --arg TAG "$STACK_TAG" '.taskDefinitionArns[] | select(contains("plandexstack" + $TAG)) | split("/")[1]')

  log "CLUSTER_NAME: $CLUSTER_NAME"
  log "SERVICE_NAME: $SERVICE_NAME"
  log "TASK_DEF_NAME: $TASK_DEF_NAME"

  # Retrieve the current task definition for the ECS service
  TASK_DEF=$(aws ecs describe-task-definition --task-definition "$TASK_DEF_NAME")
  # Extract the current task definition JSON without the revision
  TASK_DEF_JSON=$(echo $TASK_DEF | jq -c '.taskDefinition | del(.revision, .status, .taskDefinitionArn, .requiresAttributes, .compatibilities, .registeredAt, .registeredBy)')

  # Update the container image URI in the task definition JSON
  UPDATED_TASK_DEF_JSON=$(echo $TASK_DEF_JSON | jq -c --arg IMAGE_URI "$ECR_REPOSITORY:$IMAGE_TAG" '.containerDefinitions[0].image = $IMAGE_URI')

  log "UPDATED_TASK_DEF_JSON: $UPDATED_TASK_DEF_JSON"

  # Register the new task definition revision with the updated image
  TMP_FILE=$(mktemp)
  echo $UPDATED_TASK_DEF_JSON > $TMP_FILE

  REGISTERED_TASK_DEF=$(aws ecs register-task-definition --cli-input-json file://$TMP_FILE)
  NEW_TASK_DEF_ARN=$(echo $REGISTERED_TASK_DEF | jq -r '.taskDefinition.taskDefinitionArn')

  log "New task definition registered: $NEW_TASK_DEF_ARN"

  # Update the ECS service to use the new task definition revision
  SERVICE_NAME=$(aws ecs list-services --cluster "plandex-ecs-cluster-$STACK_TAG" | jq -r --arg TAG "$STACK_TAG" '.serviceArns[] | select(contains("plandex-stack-" + $TAG)) | split("/")[1]')
  aws ecs update-service --cluster "plandex-ecs-cluster-$STACK_TAG" --service "$SERVICE_NAME" --task-definition "$NEW_TASK_DEF_ARN"
  log "ECS service updated successfully with new task definition"

  log "TASK_DEF_ARN: $TASK_DEF_ARN"

  # Update the ECS service to use the new task definition
  log "Updating the ECS service with the new task definition..."
  aws ecs update-service --cluster "$CLUSTER_NAME" --service "$SERVICE_NAME" --task-definition $TASK_DEF_ARN
  log "ECS service updated successfully"

  # Clean up the temporary file
  rm $TMP_FILE
}

log "Building and pushing the Docker image to ECR..."
build_and_push_image

# Deploy or update the CloudFormation stack unless the --image-only flag is set
if [ "$SKIP_DEPLOY" = false ]; then
  log "Deploying or updating the CloudFormation stack..."
  deploy_or_update_stack
else
  log "Skipping deploy_or_update_stack due to --image-only flag"
fi

# Update the ECS service with the new Docker image if it's not a new stack
if [ "$IS_NEW_STACK" = false ]; then
  log "Updating the ECS service with the new Docker image..."
  update_ecs_service
else
  log "Skipping update_ecs_service for new stack"
fi

log "Infrastructure deployed successfully"