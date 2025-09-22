# API Documentation Infrastructure
# This file manages the OpenAPI specification and documentation resources

# S3 bucket for storing API documentation
resource "aws_s3_bucket" "api_docs" {
  bucket = "${local.project_name}-${local.environment}-api-docs"

  tags = local.common_tags
}

# S3 bucket versioning
resource "aws_s3_bucket_versioning" "api_docs" {
  bucket = aws_s3_bucket.api_docs.id
  versioning_configuration {
    status = "Enabled"
  }
}

# S3 bucket server-side encryption
resource "aws_s3_bucket_server_side_encryption_configuration" "api_docs" {
  bucket = aws_s3_bucket.api_docs.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# S3 bucket public access block
resource "aws_s3_bucket_public_access_block" "api_docs" {
  bucket = aws_s3_bucket.api_docs.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Upload OpenAPI specification to S3
resource "aws_s3_object" "openapi_spec" {
  bucket       = aws_s3_bucket.api_docs.id
  key          = "openapi.yaml"
  source       = "${path.module}/../openapi.yaml"
  content_type = "application/x-yaml"
  etag         = filemd5("${path.module}/../openapi.yaml")

  tags = merge(local.common_tags, {
    Description = "OpenAPI 3.0 specification for Lambda Go Template API"
    Version     = "1.0.0"
  })
}

# Upload OpenAPI specification as JSON (converted from YAML)
resource "aws_s3_object" "openapi_spec_json" {
  bucket       = aws_s3_bucket.api_docs.id
  key          = "openapi.json"
  content      = jsonencode(yamldecode(file("${path.module}/../openapi.yaml")))
  content_type = "application/json"
  etag         = md5(jsonencode(yamldecode(file("${path.module}/../openapi.yaml"))))

  tags = merge(local.common_tags, {
    Description = "OpenAPI 3.0 specification for Lambda Go Template API (JSON format)"
    Version     = "1.0.0"
  })
}

# CloudFront distribution for API documentation
resource "aws_cloudfront_distribution" "api_docs" {
  origin {
    domain_name = aws_s3_bucket.api_docs.bucket_regional_domain_name
    origin_id   = "S3-${aws_s3_bucket.api_docs.id}"

    s3_origin_config {
      origin_access_identity = aws_cloudfront_origin_access_identity.api_docs.cloudfront_access_identity_path
    }
  }

  enabled             = true
  is_ipv6_enabled     = true
  comment             = "API Documentation for ${local.project_name}"
  default_root_object = "index.html"

  default_cache_behavior {
    allowed_methods  = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = "S3-${aws_s3_bucket.api_docs.id}"

    forwarded_values {
      query_string = false

      cookies {
        forward = "none"
      }
    }

    viewer_protocol_policy = "redirect-to-https"
    min_ttl                = 0
    default_ttl            = 3600
    max_ttl                = 86400
  }

  # Cache behavior for API specs with longer TTL
  ordered_cache_behavior {
    path_pattern     = "*.yaml"
    allowed_methods  = ["GET", "HEAD", "OPTIONS"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = "S3-${aws_s3_bucket.api_docs.id}"

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }

    min_ttl                = 0
    default_ttl            = 86400
    max_ttl                = 31536000
    compress               = true
    viewer_protocol_policy = "redirect-to-https"
  }

  ordered_cache_behavior {
    path_pattern     = "*.json"
    allowed_methods  = ["GET", "HEAD", "OPTIONS"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = "S3-${aws_s3_bucket.api_docs.id}"

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }

    min_ttl                = 0
    default_ttl            = 86400
    max_ttl                = 31536000
    compress               = true
    viewer_protocol_policy = "redirect-to-https"
  }

  price_class = "PriceClass_100"

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  tags = local.common_tags
}

# CloudFront Origin Access Identity
resource "aws_cloudfront_origin_access_identity" "api_docs" {
  comment = "OAI for ${local.project_name} API documentation"
}

# S3 bucket policy to allow CloudFront access
resource "aws_s3_bucket_policy" "api_docs" {
  bucket = aws_s3_bucket.api_docs.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AllowCloudFrontAccess"
        Effect = "Allow"
        Principal = {
          AWS = aws_cloudfront_origin_access_identity.api_docs.iam_arn
        }
        Action   = "s3:GetObject"
        Resource = "${aws_s3_bucket.api_docs.arn}/*"
      }
    ]
  })
}

# Lambda function for generating HTML documentation
resource "aws_lambda_function" "docs_generator" {
  filename         = "${path.module}/../build/docs-generator.zip"
  function_name    = "${local.project_name}-${local.environment}-docs-generator"
  role            = aws_iam_role.docs_generator.arn
  handler         = "bootstrap"
  source_code_hash = fileexists("${path.module}/../build/docs-generator.zip") ? filebase64sha256("${path.module}/../build/docs-generator.zip") : null
  runtime         = "provided.al2023"
  architectures   = ["arm64"]
  timeout         = 30

  environment {
    variables = {
      DOCS_BUCKET = aws_s3_bucket.api_docs.id
      API_GATEWAY_URL = "https://${aws_apigatewayv2_api.api.id}.execute-api.${data.aws_region.current.region}.amazonaws.com"
    }
  }

  depends_on = [
    aws_iam_role_policy_attachment.docs_generator_basic,
    aws_iam_role_policy_attachment.docs_generator_s3,
    aws_cloudwatch_log_group.docs_generator,
  ]

  tags = local.common_tags
}

# IAM role for docs generator
resource "aws_iam_role" "docs_generator" {
  name = "${local.project_name}-${local.environment}-docs-generator"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })

  tags = local.common_tags
}

# IAM policy for docs generator S3 access
resource "aws_iam_role_policy" "docs_generator_s3" {
  name = "${local.project_name}-${local.environment}-docs-generator-s3"
  role = aws_iam_role.docs_generator.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]
        Resource = "${aws_s3_bucket.api_docs.arn}/*"
      },
      {
        Effect = "Allow"
        Action = [
          "s3:ListBucket"
        ]
        Resource = aws_s3_bucket.api_docs.arn
      }
    ]
  })
}

# Attach basic execution role to docs generator
resource "aws_iam_role_policy_attachment" "docs_generator_basic" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
  role       = aws_iam_role.docs_generator.name
}

# Attach S3 policy to docs generator
resource "aws_iam_role_policy_attachment" "docs_generator_s3" {
  policy_arn = aws_iam_policy.docs_generator_s3_policy.arn
  role       = aws_iam_role.docs_generator.name
}

# Create S3 policy resource
resource "aws_iam_policy" "docs_generator_s3_policy" {
  name        = "${local.project_name}-${local.environment}-docs-generator-s3"
  description = "S3 access for docs generator Lambda"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]
        Resource = "${aws_s3_bucket.api_docs.arn}/*"
      },
      {
        Effect = "Allow"
        Action = [
          "s3:ListBucket"
        ]
        Resource = aws_s3_bucket.api_docs.arn
      }
    ]
  })

  tags = local.common_tags
}

# CloudWatch log group for docs generator
resource "aws_cloudwatch_log_group" "docs_generator" {
  name              = "/aws/lambda/${local.project_name}-${local.environment}-docs-generator"
  retention_in_days = 14

  tags = local.common_tags
}

# S3 object for index.html (Swagger UI)
resource "aws_s3_object" "swagger_ui_index" {
  bucket       = aws_s3_bucket.api_docs.id
  key          = "index.html"
  content_type = "text/html"
  content = templatefile("${path.module}/templates/swagger-ui.html", {
    api_title        = "Lambda Go Template API"
    openapi_spec_url = "openapi.yaml"
    api_gateway_url  = "https://${aws_apigatewayv2_api.api.id}.execute-api.${data.aws_region.current.name}.amazonaws.com"
  })

  tags = merge(local.common_tags, {
    Description = "Swagger UI for API documentation"
  })
}

# Trigger to regenerate docs when OpenAPI spec changes
resource "aws_lambda_invocation" "regenerate_docs" {
  function_name = aws_lambda_function.docs_generator.function_name

  triggers = {
    openapi_hash = filemd5("${path.module}/../openapi.yaml")
  }

  input = jsonencode({
    action = "generate"
    spec_key = "openapi.yaml"
  })

  depends_on = [aws_s3_object.openapi_spec]
}
