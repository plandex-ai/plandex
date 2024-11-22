# Variable definitions
variable "environment" {
  type        = string
  description = "Deployment environment"
  default     = "development"
  validation {
    condition     = contains(["development", "staging", "production"], var.environment)
    error_message = "Environment must be development, staging, or production."
  }
}

# Local values
locals {
  common_tags = {
    Environment = var.environment
    Project     = "example"
    ManagedBy  = "terraform"
  }
  
  region_config = {
    us-east-1 = {
      instance_type = "t3.micro"
      az_count      = 2
    }
    us-west-2 = {
      instance_type = "t3.small"
      az_count      = 3
    }
  }
}

# Provider configuration
provider "aws" {
  region = "us-west-2"
  
  assume_role {
    role_arn = "arn:aws:iam::123456789012:role/terraform"
  }
  
  default_tags {
    tags = local.common_tags
  }
}

# Data source
data "aws_availability_zones" "available" {
  state = "available"
}

# Resource block with dynamic blocks
resource "aws_security_group" "example" {
  name_prefix = "example-sg"
  vpc_id      = aws_vpc.main.id
  
  dynamic "ingress" {
    for_each = var.service_ports
    content {
      from_port   = ingress.value
      to_port     = ingress.value
      protocol    = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
    }
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  lifecycle {
    create_before_destroy = true
  }
}

# Module usage
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
  
  name = "example-vpc"
  cidr = "10.0.0.0/16"
  
  azs             = data.aws_availability_zones.available.names
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24"]
  
  enable_nat_gateway = true
  single_nat_gateway = true
  
  tags = merge(
    local.common_tags,
    {
      Name = "example-vpc"
    }
  )
}

# Output values
output "vpc_id" {
  description = "The ID of the VPC"
  value       = module.vpc.vpc_id
}

output "private_subnets" {
  description = "List of private subnet IDs"
  value       = module.vpc.private_subnets
}

# Terraform settings block
terraform {
  required_version = ">= 1.0.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
  
  backend "s3" {
    bucket         = "terraform-state-example"
    key            = "example/terraform.tfstate"
    region         = "us-west-2"
    encrypt        = true
    dynamodb_table = "terraform-locks"
  }
}
