# Serverless Infrastructure - Go Lambda Template
# Clean, modular serverless architecture with DynamoDB, EventBridge, and API Gateway

terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.14.0"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# Local values for consistent naming and tagging
locals {
  namespace_prefix = var.namespace != "" ? "${var.namespace}-" : ""
  function_name    = "${local.namespace_prefix}${var.function_name}"

  common_tags = {
    Project     = "lambda-go-template"
    Environment = var.environment
    ManagedBy   = "terraform"
    Namespace   = var.namespace
  }
}

# ========================================
# DYNAMODB TABLES
# ========================================

# Users table for CRUD operations
resource "aws_dynamodb_table" "users" {
  name         = "${local.function_name}-users"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }

  attribute {
    name = "email"
    type = "S"
  }

  global_secondary_index {
    name            = "email-index"
    hash_key        = "email"
    projection_type = "ALL"
  }

  server_side_encryption {
    enabled = true
  }

  point_in_time_recovery {
    enabled = true
  }

  tags = merge(local.common_tags, {
    Resource = "dynamodb-users"
  })
}

# Posts table for CRUD operations
resource "aws_dynamodb_table" "posts" {
  name         = "${local.function_name}-posts"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"
  range_key    = "created_at"

  attribute {
    name = "id"
    type = "S"
  }

  attribute {
    name = "created_at"
    type = "S"
  }

  attribute {
    name = "user_id"
    type = "S"
  }

  global_secondary_index {
    name            = "user-posts-index"
    hash_key        = "user_id"
    range_key       = "created_at"
    projection_type = "ALL"
  }

  server_side_encryption {
    enabled = true
  }

  point_in_time_recovery {
    enabled = true
  }

  tags = merge(local.common_tags, {
    Resource = "dynamodb-posts"
  })
}

# Audit logs table for EventBridge events
resource "aws_dynamodb_table" "audit_logs" {
  name         = "${local.function_name}-audit-logs"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "event_id"
  range_key    = "timestamp"

  attribute {
    name = "event_id"
    type = "S"
  }

  attribute {
    name = "timestamp"
    type = "S"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  server_side_encryption {
    enabled = true
  }

  tags = merge(local.common_tags, {
    Resource = "dynamodb-audit"
  })
}

# ========================================
# EVENTBRIDGE
# ========================================

# Custom event bus for application events
resource "aws_cloudwatch_event_bus" "app_events" {
  name = "${local.function_name}-events"

  tags = merge(local.common_tags, {
    Resource = "eventbridge-bus"
  })
}

# Event rules for capturing CRUD operations
resource "aws_cloudwatch_event_rule" "crud_events" {
  name           = "${local.function_name}-crud-events"
  description    = "Capture all CRUD events for audit logging"
  event_bus_name = aws_cloudwatch_event_bus.app_events.name

  event_pattern = jsonencode({
    source = ["lambda.${local.function_name}"]
    detail-type = [
      "User Created", "User Updated", "User Deleted",
      "Post Created", "Post Updated", "Post Deleted"
    ]
  })

  tags = local.common_tags
}

# ========================================
# AWS APPCONFIG - Feature Flags & Configuration
# ========================================

# AppConfig Application
resource "aws_appconfig_application" "main" {
  name        = "${local.function_name}-config"
  description = "Feature flags and configuration for ${local.function_name}"

  tags = merge(local.common_tags, {
    Resource = "appconfig-application"
  })
}

# Feature Flags Configuration Profile
resource "aws_appconfig_configuration_profile" "feature_flags" {
  application_id = aws_appconfig_application.main.id
  name           = "feature-flags"
  description    = "Feature flags for dynamic behavior control"
  location_uri   = "hosted"
  type           = "AWS.AppConfig.FeatureFlags"

  tags = local.common_tags
}

# Application Configuration Profile
resource "aws_appconfig_configuration_profile" "app_config" {
  application_id = aws_appconfig_application.main.id
  name           = "application-config"
  description    = "Application configuration settings"
  location_uri   = "hosted"

  validator {
    content = jsonencode({
      "$schema" = "http://json-schema.org/draft-07/schema#"
      type      = "object"
      properties = {
        database = {
          type = "object"
          properties = {
            connectionTimeout = { type = "integer", minimum = 1000, maximum = 30000 }
            retryAttempts     = { type = "integer", minimum = 1, maximum = 5 }
          }
          required = ["connectionTimeout", "retryAttempts"]
        }
        api = {
          type = "object"
          properties = {
            rateLimit     = { type = "integer", minimum = 10, maximum = 1000 }
            timeout       = { type = "integer", minimum = 1000, maximum = 30000 }
            enableCaching = { type = "boolean" }
          }
          required = ["rateLimit", "timeout", "enableCaching"]
        }
        features = {
          type = "object"
          properties = {
            enableAdvancedLogging = { type = "boolean" }
            enableMetrics         = { type = "boolean" }
            maintenanceMode       = { type = "boolean" }
          }
          required = ["enableAdvancedLogging", "enableMetrics", "maintenanceMode"]
        }
      }
      required = ["database", "api", "features"]
    })
    type = "JSON_SCHEMA"
  }

  tags = local.common_tags
}

# Ephemeral Environment (based on namespace)
resource "aws_appconfig_environment" "ephemeral" {
  count          = var.namespace != "" ? 1 : 0
  name           = var.namespace
  description    = "Ephemeral environment configuration for ${var.namespace}"
  application_id = aws_appconfig_application.main.id

  monitor {
    alarm_arn      = aws_cloudwatch_metric_alarm.lambda_error_rate.arn
    alarm_role_arn = aws_iam_role.appconfig_monitor.arn
  }

  tags = local.common_tags
}

# Production Environment
resource "aws_appconfig_environment" "production" {
  name           = "production"
  description    = "Production environment configuration"
  application_id = aws_appconfig_application.main.id

  monitor {
    alarm_arn      = aws_cloudwatch_metric_alarm.lambda_error_rate.arn
    alarm_role_arn = aws_iam_role.appconfig_monitor.arn
  }

  tags = local.common_tags
}

# Deployment Strategy - Canary
resource "aws_appconfig_deployment_strategy" "canary" {
  name                           = "${local.function_name}-canary"
  description                    = "Canary deployment strategy for safe rollouts"
  deployment_duration_in_minutes = 10
  final_bake_time_in_minutes     = 10
  growth_factor                  = 20
  growth_type                    = "EXPONENTIAL"
  replicate_to                   = "NONE"

  tags = local.common_tags
}

# IAM Role for AppConfig Monitoring
resource "aws_iam_role" "appconfig_monitor" {
  name = "${local.function_name}-appconfig-monitor"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "appconfig.amazonaws.com"
        }
      }
    ]
  })

  tags = local.common_tags
}

resource "aws_iam_role_policy" "appconfig_monitor" {
  name = "${local.function_name}-appconfig-monitor"
  role = aws_iam_role.appconfig_monitor.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "cloudwatch:DescribeAlarms"
        ]
        Resource = "*"
      }
    ]
  })
}

# CloudWatch Alarm for Error Rate Monitoring
resource "aws_cloudwatch_metric_alarm" "lambda_error_rate" {
  alarm_name          = "${local.function_name}-error-rate"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "Errors"
  namespace           = "AWS/Lambda"
  period              = "300"
  statistic           = "Sum"
  threshold           = "5"
  alarm_description   = "This metric monitors lambda error rate"
  alarm_actions       = []

  dimensions = {
    FunctionName = module.users_lambda.lambda_function_name
  }

  tags = local.common_tags
}

# Initial Feature Flags Configuration
resource "aws_appconfig_hosted_configuration_version" "feature_flags_v1" {
  application_id           = aws_appconfig_application.main.id
  configuration_profile_id = aws_appconfig_configuration_profile.feature_flags.configuration_profile_id
  description              = "Initial feature flags configuration"
  content_type             = "application/json"

  content = jsonencode({
    flags = {
      enableNewUserFlow = {
        enabled = false
      }
      enableAdvancedSearch = {
        enabled = false
      }
      enableExperimentalFeatures = {
        enabled = false
      }
      maintenanceMode = {
        enabled = false
      }
      enableDetailedLogging = {
        enabled = true
      }
    }
    values = {
      enableNewUserFlow = {
        enabled = false
      }
      enableAdvancedSearch = {
        enabled = false
      }
      enableExperimentalFeatures = {
        enabled = false
      }
      maintenanceMode = {
        enabled = false
      }
      enableDetailedLogging = {
        enabled = true
      }
    }
    version = "1"
  })
}

# Initial Application Configuration
resource "aws_appconfig_hosted_configuration_version" "app_config_v1" {
  application_id           = aws_appconfig_application.main.id
  configuration_profile_id = aws_appconfig_configuration_profile.app_config.configuration_profile_id
  description              = "Initial application configuration"
  content_type             = "application/json"

  content = jsonencode({
    database = {
      connectionTimeout = 5000
      retryAttempts     = 3
    }
    api = {
      rateLimit     = 100
      timeout       = 10000
      enableCaching = true
    }
    features = {
      enableAdvancedLogging = true
      enableMetrics         = true
      maintenanceMode       = false
    }
  })
}

# ========================================
# LAMBDA FUNCTIONS
# ========================================

# Health check Lambda
module "health_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 8.1.0"

  function_name = "${local.function_name}-health"
  description   = "Health check endpoint"
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  memory_size   = 256
  timeout       = 15

  create_package         = false
  local_existing_package = "../build/health.zip"

  environment_variables = {
    ENVIRONMENT              = var.environment
    CGO_ENABLED              = "0"
    GOOS                     = "linux"
    GOARCH                   = "arm64"
    APPCONFIG_APPLICATION_ID = aws_appconfig_application.main.id
    APPCONFIG_ENVIRONMENT    = var.namespace != "" ? var.namespace : "production"
    APPCONFIG_FEATURE_FLAGS  = aws_appconfig_configuration_profile.feature_flags.name
    APPCONFIG_APP_CONFIG     = aws_appconfig_configuration_profile.app_config.name
  }

  attach_cloudwatch_logs_policy     = true
  cloudwatch_logs_retention_in_days = 14

  attach_policy_statements = true
  policy_statements = {
    appconfig = {
      effect = "Allow"
      actions = [
        "appconfig:StartConfigurationSession",
        "appconfig:GetConfiguration"
      ]
      resources = ["*"]
    }
  }

  tags = local.common_tags
}

# Users CRUD Lambda
module "users_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 8.1.0"

  function_name = "${local.function_name}-users"
  description   = "Users CRUD API"
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  memory_size   = 512
  timeout       = 30

  create_package         = false
  local_existing_package = "../build/users.zip"

  tracing_mode = "Active"

  environment_variables = {
    ENVIRONMENT              = var.environment
    CGO_ENABLED              = "0"
    GOOS                     = "linux"
    GOARCH                   = "arm64"
    USERS_TABLE_NAME         = aws_dynamodb_table.users.name
    EVENT_BUS_NAME           = aws_cloudwatch_event_bus.app_events.name
    APPCONFIG_APPLICATION_ID = aws_appconfig_application.main.id
    APPCONFIG_ENVIRONMENT    = var.namespace != "" ? var.namespace : "production"
    APPCONFIG_FEATURE_FLAGS  = aws_appconfig_configuration_profile.feature_flags.name
    APPCONFIG_APP_CONFIG     = aws_appconfig_configuration_profile.app_config.name
  }

  attach_cloudwatch_logs_policy     = true
  attach_tracing_policy             = true
  cloudwatch_logs_retention_in_days = 14

  attach_policy_statements = true
  policy_statements = {
    dynamodb = {
      effect = "Allow"
      actions = [
        "dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:UpdateItem",
        "dynamodb:DeleteItem", "dynamodb:Query", "dynamodb:Scan"
      ]
      resources = [aws_dynamodb_table.users.arn, "${aws_dynamodb_table.users.arn}/*"]
    }
    eventbridge = {
      effect    = "Allow"
      actions   = ["events:PutEvents"]
      resources = [aws_cloudwatch_event_bus.app_events.arn]
    }
    appconfig = {
      effect = "Allow"
      actions = [
        "appconfig:StartConfigurationSession",
        "appconfig:GetConfiguration"
      ]
      resources = ["*"]
    }
  }

  tags = local.common_tags
}

# Posts CRUD Lambda
module "posts_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 8.1.0"

  function_name = "${local.function_name}-posts"
  description   = "Posts CRUD API"
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  memory_size   = 512
  timeout       = 30

  create_package         = false
  local_existing_package = "../build/posts.zip"

  tracing_mode = "Active"

  environment_variables = {
    ENVIRONMENT              = var.environment
    CGO_ENABLED              = "0"
    GOOS                     = "linux"
    GOARCH                   = "arm64"
    POSTS_TABLE_NAME         = aws_dynamodb_table.posts.name
    USERS_TABLE_NAME         = aws_dynamodb_table.users.name
    EVENT_BUS_NAME           = aws_cloudwatch_event_bus.app_events.name
    APPCONFIG_APPLICATION_ID = aws_appconfig_application.main.id
    APPCONFIG_ENVIRONMENT    = var.namespace != "" ? var.namespace : "production"
    APPCONFIG_FEATURE_FLAGS  = aws_appconfig_configuration_profile.feature_flags.name
    APPCONFIG_APP_CONFIG     = aws_appconfig_configuration_profile.app_config.name
  }

  attach_cloudwatch_logs_policy     = true
  attach_tracing_policy             = true
  cloudwatch_logs_retention_in_days = 14

  attach_policy_statements = true
  policy_statements = {
    dynamodb = {
      effect = "Allow"
      actions = [
        "dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:UpdateItem",
        "dynamodb:DeleteItem", "dynamodb:Query", "dynamodb:Scan"
      ]
      resources = [
        aws_dynamodb_table.posts.arn, "${aws_dynamodb_table.posts.arn}/*",
        aws_dynamodb_table.users.arn, "${aws_dynamodb_table.users.arn}/*"
      ]
    }
    eventbridge = {
      effect    = "Allow"
      actions   = ["events:PutEvents"]
      resources = [aws_cloudwatch_event_bus.app_events.arn]
    }
    appconfig = {
      effect = "Allow"
      actions = [
        "appconfig:StartConfigurationSession",
        "appconfig:GetConfiguration"
      ]
      resources = ["*"]
    }
  }

  tags = local.common_tags
}

# Event processor Lambda for audit logging
module "event_processor_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 8.1.0"

  function_name = "${local.function_name}-event-processor"
  description   = "Process EventBridge events for audit logging"
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  memory_size   = 256
  timeout       = 30

  create_package         = false
  local_existing_package = "../build/event-processor.zip"

  environment_variables = {
    ENVIRONMENT              = var.environment
    CGO_ENABLED              = "0"
    GOOS                     = "linux"
    GOARCH                   = "arm64"
    AUDIT_TABLE_NAME         = aws_dynamodb_table.audit_logs.name
    APPCONFIG_APPLICATION_ID = aws_appconfig_application.main.id
    APPCONFIG_ENVIRONMENT    = var.namespace != "" ? var.namespace : "production"
    APPCONFIG_FEATURE_FLAGS  = aws_appconfig_configuration_profile.feature_flags.name
    APPCONFIG_APP_CONFIG     = aws_appconfig_configuration_profile.app_config.name
  }

  attach_cloudwatch_logs_policy     = true
  cloudwatch_logs_retention_in_days = 14

  attach_policy_statements = true
  policy_statements = {
    dynamodb = {
      effect    = "Allow"
      actions   = ["dynamodb:PutItem", "dynamodb:UpdateItem"]
      resources = [aws_dynamodb_table.audit_logs.arn]
    }
    appconfig = {
      effect = "Allow"
      actions = [
        "appconfig:StartConfigurationSession",
        "appconfig:GetConfiguration"
      ]
      resources = ["*"]
    }
  }

  tags = local.common_tags
}

# ========================================
# API GATEWAY
# ========================================

module "api_gateway" {
  source  = "terraform-aws-modules/apigateway-v2/aws"
  version = "~> 5.3.1"

  name          = "${local.function_name}-api"
  description   = "Serverless CRUD API"
  protocol_type = "HTTP"

  create_domain_name    = false
  create_domain_records = false
  create_certificate    = false

  cors_configuration = {
    allow_headers     = ["content-type", "authorization", "x-api-key"]
    allow_methods     = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allow_origins     = ["https://localhost:3000", "https://example.com"]
    expose_headers    = ["date", "keep-alive"]
    max_age           = 300
    allow_credentials = false
  }

  routes = {
    # Health check
    "GET /health" = {
      integration = {
        uri                    = module.health_lambda.lambda_function_arn
        payload_format_version = "2.0"
        timeout_milliseconds   = 12000
      }
    }

    # Users CRUD
    "GET /users"         = { integration = { uri = module.users_lambda.lambda_function_arn, payload_format_version = "2.0" } }
    "GET /users/{id}"    = { integration = { uri = module.users_lambda.lambda_function_arn, payload_format_version = "2.0" } }
    "POST /users"        = { integration = { uri = module.users_lambda.lambda_function_arn, payload_format_version = "2.0" } }
    "PUT /users/{id}"    = { integration = { uri = module.users_lambda.lambda_function_arn, payload_format_version = "2.0" } }
    "DELETE /users/{id}" = { integration = { uri = module.users_lambda.lambda_function_arn, payload_format_version = "2.0" } }

    # Posts CRUD
    "GET /posts"         = { integration = { uri = module.posts_lambda.lambda_function_arn, payload_format_version = "2.0" } }
    "GET /posts/{id}"    = { integration = { uri = module.posts_lambda.lambda_function_arn, payload_format_version = "2.0" } }
    "POST /posts"        = { integration = { uri = module.posts_lambda.lambda_function_arn, payload_format_version = "2.0" } }
    "PUT /posts/{id}"    = { integration = { uri = module.posts_lambda.lambda_function_arn, payload_format_version = "2.0" } }
    "DELETE /posts/{id}" = { integration = { uri = module.posts_lambda.lambda_function_arn, payload_format_version = "2.0" } }
  }

  tags = local.common_tags
}

# ========================================
# LAMBDA PERMISSIONS
# ========================================

# API Gateway permissions
resource "aws_lambda_permission" "api_gateway_health" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = module.health_lambda.lambda_function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${module.api_gateway.api_execution_arn}/*/*"
}

resource "aws_lambda_permission" "api_gateway_users" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = module.users_lambda.lambda_function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${module.api_gateway.api_execution_arn}/*/*"
}

resource "aws_lambda_permission" "api_gateway_posts" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = module.posts_lambda.lambda_function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${module.api_gateway.api_execution_arn}/*/*"
}

# EventBridge permissions
resource "aws_cloudwatch_event_target" "audit_target" {
  rule           = aws_cloudwatch_event_rule.crud_events.name
  event_bus_name = aws_cloudwatch_event_bus.app_events.name
  target_id      = "AuditProcessor"
  arn            = module.event_processor_lambda.lambda_function_arn
}

resource "aws_lambda_permission" "allow_eventbridge" {
  statement_id  = "AllowExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = module.event_processor_lambda.lambda_function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.crud_events.arn
}