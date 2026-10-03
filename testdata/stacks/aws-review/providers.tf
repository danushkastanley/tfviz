# Synthetic fixture stack. Never point this configuration at a real AWS account.
# scripts/fixturegen supplies deliberately fake credentials through the
# producer's environment and every validation call is skipped, so
# `plan -refresh=false` runs without contacting AWS.

terraform {
  required_version = ">= 1.10"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "6.67.0"
    }
  }
}

provider "aws" {
  region = "eu-west-1"

  skip_credentials_validation = true
  skip_requesting_account_id  = true
  skip_metadata_api_check     = true
  skip_region_validation      = true
}

variable "phase" {
  description = "Which variant of the stack to plan: \"prior\" builds the recorded state, \"proposed\" is the change under review."
  type        = string

  validation {
    condition     = contains(["prior", "proposed"], var.phase)
    error_message = "phase must be \"prior\" or \"proposed\"."
  }
}

variable "db_password" {
  description = "Synthetic canary password."
  type        = string
  sensitive   = true
}

variable "kafka_password" {
  description = "Synthetic canary SCRAM password."
  type        = string
  sensitive   = true
}

locals {
  proposed = var.phase == "proposed"
}
