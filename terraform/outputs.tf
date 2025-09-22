output "hello_lambda_function_arn" {
  description = "ARN of the Hello Lambda function"
  value       = module.lambda_functions["hello"].lambda_function_arn
}

output "users_lambda_function_arn" {
  description = "ARN of the Users Lambda function"
  value       = module.lambda_functions["users"].lambda_function_arn
}

output "api_gateway_url" {
  description = "URL of the API Gateway"
  value       = aws_apigatewayv2_api.api.api_endpoint
}

output "hello_endpoint" {
  description = "Hello endpoint URL"
  value       = "${aws_apigatewayv2_api.api.api_endpoint}/prod/hello"
}

output "users_endpoint" {
  description = "Users endpoint URL"
  value       = "${aws_apigatewayv2_api.api.api_endpoint}/prod/users"
}

output "users_table_name" {
  description = "Name of the Users DynamoDB table"
  value       = aws_dynamodb_table.users.name
}

output "audit_logs_table_name" {
  description = "Name of the Audit Logs DynamoDB table"
  value       = aws_dynamodb_table.audit_logs.name
}

output "event_bus_name" {
  description = "Name of the custom EventBridge bus"
  value       = aws_cloudwatch_event_bus.app_events.name
}
