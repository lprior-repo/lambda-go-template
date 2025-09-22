variable "aws_region" {
  description = "AWS region for resources"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Name of the project"
  type        = string
  default     = "lambda-go-template"
}

variable "function_name" {
  description = "Base name for Lambda functions"
  type        = string
  default     = "go-lambda"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "dev"
}

variable "namespace" {
  description = "Namespace for resource naming (enables ephemeral infrastructure)"
  type        = string
  default     = ""
  validation {
    condition     = can(regex("^[a-z0-9-]*$", var.namespace))
    error_message = "Namespace must contain only lowercase letters, numbers, and hyphens."
  }
}

variable "is_ephemeral" {
  description = "Whether this is an ephemeral environment (for testing/development)"
  type        = bool
  default     = false
}

variable "github_org" {
  description = "GitHub organization name (for OIDC setup)"
  type        = string
  default     = ""
}

variable "github_repo" {
  description = "GitHub repository name (for OIDC setup)"
  type        = string
  default     = ""
}

variable "create_oidc_provider" {
  description = "Whether to create GitHub OIDC provider"
  type        = bool
  default     = false
}
