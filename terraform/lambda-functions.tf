# Lambda function modules for each endpoint
module "lambda_functions" {
  source   = "terraform-aws-modules/lambda/aws"
  version  = "~> 8.1"
  for_each = local.lambda_functions

  function_name = each.value.name
  description   = "Serverless function for ${each.key} endpoint"
  handler       = each.value.handler
  runtime       = each.value.runtime
  architectures = ["arm64"]

  create_package         = false
  local_existing_package = each.value.source_dir

  timeout     = 30
  memory_size = 512

  environment_variables = {
    ENVIRONMENT      = local.environment
    LOG_LEVEL        = "info"
    USERS_TABLE_NAME = aws_dynamodb_table.users.name
    EVENT_BUS_NAME   = aws_cloudwatch_event_bus.app_events.name
  }

  # CloudWatch Logs
  attach_cloudwatch_logs_policy     = true
  cloudwatch_logs_retention_in_days = 14

  # X-Ray tracing
  tracing_mode          = "Active"
  attach_tracing_policy = true

  # DynamoDB permissions
  attach_policy_statements = true
  policy_statements = {
    dynamodb = {
      effect = "Allow"
      actions = [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem",
        "dynamodb:DeleteItem",
        "dynamodb:Query",
        "dynamodb:Scan"
      ]
      resources = [
        aws_dynamodb_table.users.arn,
        "${aws_dynamodb_table.users.arn}/*",
        aws_dynamodb_table.audit_logs.arn,
        "${aws_dynamodb_table.audit_logs.arn}/*"
      ]
    }
    eventbridge = {
      effect    = "Allow"
      actions   = ["events:PutEvents"]
      resources = [aws_cloudwatch_event_bus.app_events.arn]
    }
  }

  tags = local.common_tags
}

# Lambda permissions for API Gateway
resource "aws_lambda_permission" "api_gateway_lambda" {
  for_each = local.lambda_functions

  statement_id  = "AllowExecutionFromAPIGateway-${each.key}"
  action        = "lambda:InvokeFunction"
  function_name = module.lambda_functions[each.key].lambda_function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.api.execution_arn}/*/*"
}
