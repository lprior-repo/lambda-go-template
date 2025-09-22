locals {
  # Configuration
  aws_region     = var.aws_region
  project_name   = var.project_name
  environment    = var.is_ephemeral ? "ephemeral" : var.environment
  namespace      = var.namespace
  ephemeral_environment = var.is_ephemeral

  # Computed values
  actual_namespace   = local.namespace != "" ? local.namespace : local.environment
  function_base_name = "${local.project_name}-${local.actual_namespace}"

  # Lambda functions configuration for Go template
  lambda_functions = {
    hello = {
      name        = "${local.function_base_name}-hello"
      source_dir  = "../build/hello.zip"
      runtime     = "provided.al2023"
      handler     = "bootstrap"
      routes      = [{ path = "/hello", method = "GET", auth = false }]
    }
    users = {
      name        = "${local.function_base_name}-users"
      source_dir  = "../build/users.zip"
      runtime     = "provided.al2023"
      handler     = "bootstrap"
      routes      = [{ path = "/users", method = "ANY", auth = false }]
    }
  }

  # Common tags
  common_tags = {
    Project     = local.project_name
    Environment = local.environment
    Namespace   = local.actual_namespace
    ManagedBy   = "terraform"
    Ephemeral   = local.ephemeral_environment ? "true" : "false"
  }
}
