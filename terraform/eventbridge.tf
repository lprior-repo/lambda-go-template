# EventBridge Custom Bus
resource "aws_cloudwatch_event_bus" "app_events" {
  name = "${local.function_base_name}-events"

  tags = local.common_tags
}

# Event rules for capturing CRUD operations (for future use)
resource "aws_cloudwatch_event_rule" "crud_events" {
  name           = "${local.function_base_name}-crud-events"
  description    = "Capture all CRUD events for audit logging"
  event_bus_name = aws_cloudwatch_event_bus.app_events.name

  event_pattern = jsonencode({
    source      = ["lambda.${local.function_base_name}"]
    detail-type = [
      "User Created", "User Updated", "User Deleted"
    ]
  })

  state = "DISABLED"  # Disabled until we have an event processor

  tags = local.common_tags
}
