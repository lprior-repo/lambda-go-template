# SAM Local Debugging for Terraform-based Go Lambda Functions

This guide explains how to set up AWS SAM CLI for local development, testing, and debugging of Go Lambda functions that are defined using Terraform configuration.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Project Structure](#project-structure)
- [SAM Configuration](#sam-configuration)
- [Local Development Workflow](#local-development-workflow)
- [Debugging Setup](#debugging-setup)
- [IDE Integration](#ide-integration)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Overview

AWS SAM CLI now supports local testing and debugging of serverless applications defined using HashiCorp Terraform. This integration allows you to:

- Run Lambda functions locally
- Test API Gateway integrations
- Debug functions with breakpoints
- Simulate AWS services locally
- Test with real AWS resources

The key to this integration is the `--hook-name terraform` flag, which tells SAM to read Terraform configuration instead of SAM templates.

## Prerequisites

Before you can use SAM for local debugging, ensure you have:

1. **AWS CLI** installed and configured with valid credentials
2. **HashiCorp Terraform** (version 0.12+)
3. **AWS SAM CLI** (latest version)
4. **Docker** (required for local Lambda execution)
5. **Go** (1.21+)
6. **Delve debugger** for Go debugging: `go install github.com/go-delve/delve/cmd/dlv@latest`

### Installation Commands

```bash
# Install SAM CLI (macOS with Homebrew)
brew install aws-sam-cli

# Install SAM CLI (Linux/Windows - download from AWS)
# https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html

# Install Delve debugger
go install github.com/go-delve/delve/cmd/dlv@latest

# Verify installations
sam --version
dlv version
docker --version
```

## Project Structure

Our project structure supports SAM local debugging:

```
lambda-go-template/
├── src/
│   ├── hello/
│   │   └── main.go           # Lambda function source
│   └── users/
│       └── main.go           # Lambda function source
├── build/
│   ├── hello.zip             # Built deployment package
│   └── users.zip             # Built deployment package
├── terraform/
│   ├── lambda-functions.tf   # Lambda function definitions
│   ├── locals.tf            # Lambda configuration
│   └── ...
├── events/                   # Test event payloads
│   ├── hello-event.json
│   └── users-event.json
└── .vscode/
    └── launch.json          # VS Code debugging configuration
```

## SAM Configuration

### 1. Enable Beta Features

SAM Terraform support is currently in preview. Enable it by either:

**Option A: Command line flag**
```bash
sam build --hook-name terraform --beta-features
```

**Option B: Configuration file (`samconfig.toml`)**
```toml
version = 0.1

[default.global.parameters]
beta_features = true

[default.build.parameters]
hook_name = "terraform"

[default.local_invoke.parameters]
hook_name = "terraform"

[default.local_start_api.parameters]
hook_name = "terraform"
```

### 2. Terraform Metadata Configuration

For SAM to understand your Terraform-defined Lambda functions, you need to add metadata resources. Add this to your `terraform/lambda-functions.tf`:

```hcl
# SAM metadata for local debugging
resource "null_resource" "sam_metadata_hello" {
  triggers = {
    resource_name         = "aws_lambda_function.hello"
    resource_type        = "ZIP_LAMBDA_FUNCTION"
    original_source_code = "../src/hello"
    built_output_path    = "../build/hello.zip"
  }
}

resource "null_resource" "sam_metadata_users" {
  triggers = {
    resource_name         = "aws_lambda_function.users"
    resource_type        = "ZIP_LAMBDA_FUNCTION"
    original_source_code = "../src/users"
    built_output_path    = "../build/users.zip"
  }
}
```

**Note**: If using the community `terraform-aws-lambda` module (version 4.0+), SAM metadata is automatically generated.

## Local Development Workflow

### 1. Build Lambda Functions

First, build your Go Lambda functions:

```bash
# Build all functions
make build

# Or build manually
mkdir -p build
cd src/hello
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -gcflags="all=-N -l" -o bootstrap main.go
zip -r ../../build/hello.zip bootstrap
rm bootstrap
cd ../..
```

**Important**: Use `-gcflags="all=-N -l"` to disable optimizations for debugging.

### 2. Build SAM Application

```bash
# Navigate to terraform directory
cd terraform

# Build SAM application from Terraform
sam build --hook-name terraform --beta-features
```

### 3. Local Invocation

Test individual Lambda functions:

```bash
# Invoke hello function with test event
sam local invoke \
  --hook-name terraform \
  --beta-features \
  aws_lambda_function.hello \
  -e ../events/hello-event.json

# Invoke users function
sam local invoke \
  --hook-name terraform \
  --beta-features \
  aws_lambda_function.users \
  -e ../events/users-event.json
```

### 4. Local API Gateway

Start a local API Gateway to test HTTP endpoints:

```bash
# Start local API Gateway
sam local start-api \
  --hook-name terraform \
  --beta-features \
  --port 3000

# Test endpoints
curl http://localhost:3000/hello
curl http://localhost:3000/users
```

### 5. Local Lambda Service

Emulate AWS Lambda service locally:

```bash
# Start local Lambda service
sam local start-lambda \
  --hook-name terraform \
  --beta-features

# Invoke using AWS CLI
aws lambda invoke \
  --function-name aws_lambda_function.hello \
  --endpoint-url http://127.0.0.1:3001/ \
  --payload file://events/hello-event.json \
  response.json
```

## Debugging Setup

### Debug Mode Invocation

To debug Lambda functions, use the `--debug-port` flag:

```bash
# Debug hello function on port 5986 (Delve default)
sam local invoke \
  --hook-name terraform \
  --beta-features \
  --debug-port 5986 \
  aws_lambda_function.hello \
  -e ../events/hello-event.json

# Debug with API Gateway
sam local start-api \
  --hook-name terraform \
  --beta-features \
  --debug-port 5986
```

### Preparing Go Code for Debugging

1. **Add debug imports** (optional, for enhanced debugging):

```go
package main

import (
    // Standard imports...
    "log"

    // Uncomment for debugging
    // _ "github.com/go-delve/delve/service/debugger"
)

func main() {
    // Uncomment for debugging entry point
    // log.Println("Lambda function starting - attach debugger now")
    // time.Sleep(10 * time.Second)

    // Your existing code...
}
```

2. **Build with debug symbols**:

```bash
# Build with debugging enabled
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
  -gcflags="all=-N -l" \
  -o bootstrap main.go
```

## IDE Integration

### Visual Studio Code

Create `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Lambda Hello",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "/var/task",
            "port": 5986,
            "host": "127.0.0.1",
            "showLog": true,
            "trace": "log",
            "logOutput": "rpc"
        },
        {
            "name": "Debug Lambda Users",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "/var/task",
            "port": 5986,
            "host": "127.0.0.1",
            "showLog": true,
            "trace": "log",
            "logOutput": "rpc"
        }
    ]
}
```

**Debugging Steps**:

1. Set breakpoints in your Go code
2. Start SAM in debug mode:
   ```bash
   sam local invoke --debug-port 5986 --hook-name terraform --beta-features aws_lambda_function.hello -e ../events/hello-event.json
   ```
3. In VS Code, go to Run and Debug panel
4. Select "Debug Lambda Hello" and click Start
5. The debugger will attach and hit your breakpoints

### GoLand/IntelliJ

Create a "Go Remote" debug configuration:

1. Go to **Run** → **Edit Configurations**
2. Add **Go Remote** configuration
3. Set **Host**: `localhost`, **Port**: `5986`
4. Start debugging the same way as VS Code

### Command Line Debugging

Use Delve directly:

```bash
# Start SAM in debug mode
sam local invoke --debug-port 5986 --hook-name terraform --beta-features aws_lambda_function.hello -e ../events/hello-event.json

# In another terminal, connect with Delve
dlv connect localhost:5986
```

Delve commands:
```
(dlv) break main.main
(dlv) continue
(dlv) step
(dlv) next
(dlv) print variableName
(dlv) locals
```

## Test Event Examples

Create test events in the `events/` directory:

### events/hello-event.json
```json
{
  "version": "2.0",
  "routeKey": "GET /hello",
  "rawPath": "/hello",
  "rawQueryString": "",
  "headers": {
    "accept": "application/json",
    "content-length": "0",
    "host": "localhost:3000",
    "user-agent": "curl/7.64.1",
    "x-forwarded-port": "3000",
    "x-forwarded-proto": "http"
  },
  "requestContext": {
    "accountId": "123456789012",
    "apiId": "1234567890",
    "domainName": "localhost:3000",
    "http": {
      "method": "GET",
      "path": "/hello",
      "protocol": "HTTP/1.1",
      "sourceIp": "127.0.0.1",
      "userAgent": "curl/7.64.1"
    },
    "requestId": "test-request-id",
    "routeKey": "GET /hello",
    "stage": "$default",
    "time": "12/Mar/2023:19:03:58 +0000",
    "timeEpoch": 1678651438123
  },
  "isBase64Encoded": false
}
```

### events/users-event.json
```json
{
  "version": "2.0",
  "routeKey": "GET /users",
  "rawPath": "/users",
  "rawQueryString": "",
  "headers": {
    "accept": "application/json",
    "content-length": "0",
    "host": "localhost:3000"
  },
  "requestContext": {
    "accountId": "123456789012",
    "apiId": "1234567890",
    "domainName": "localhost:3000",
    "http": {
      "method": "GET",
      "path": "/users",
      "protocol": "HTTP/1.1",
      "sourceIp": "127.0.0.1"
    },
    "requestId": "test-request-id",
    "routeKey": "GET /users",
    "stage": "$default"
  },
  "isBase64Encoded": false
}
```

## Troubleshooting

### Common Issues

1. **"No template file found"**
   - Ensure you're in the `terraform/` directory
   - Use `--hook-name terraform` flag
   - Enable `--beta-features`

2. **"Function not found"**
   - Check resource names in Terraform match SAM invoke commands
   - Verify null_resource metadata is correct
   - Run `sam build` before invoking

3. **Docker permission errors**
   - Ensure Docker is running
   - Check Docker permissions (add user to docker group on Linux)

4. **Debugger won't connect**
   - Verify port 5986 is available
   - Check firewall settings
   - Ensure Go binary has debug symbols (`-gcflags="all=-N -l"`)

5. **Environment variables not working**
   - SAM reads environment variables from Terraform configuration
   - Local testing may not have access to AWS resources
   - Use `--env-vars` for custom local environment variables

### Debug Output

Enable verbose logging:

```bash
# Verbose SAM output
sam local invoke --debug --hook-name terraform --beta-features aws_lambda_function.hello

# Docker container logs
docker logs $(docker ps -q --filter ancestor=public.ecr.aws/sam/emulation-provided)
```

### Local Environment Variables

Create `env.json` for local testing:

```json
{
  "hello": {
    "ENVIRONMENT": "local",
    "LOG_LEVEL": "debug",
    "USERS_TABLE_NAME": "lambda-go-template-dev-users",
    "EVENT_BUS_NAME": "lambda-go-template-dev-events"
  },
  "users": {
    "ENVIRONMENT": "local",
    "LOG_LEVEL": "debug",
    "USERS_TABLE_NAME": "lambda-go-template-dev-users",
    "EVENT_BUS_NAME": "lambda-go-template-dev-events"
  }
}
```

Use with SAM:
```bash
sam local invoke --env-vars env.json --hook-name terraform --beta-features aws_lambda_function.hello
```

## Best Practices

### 1. Development Workflow

```bash
# Recommended development cycle
make build                    # Build Lambda functions
cd terraform
sam build --hook-name terraform --beta-features  # Build SAM app
sam local start-api --hook-name terraform --beta-features  # Start local API
```

### 2. Code Organization

- Keep debug-specific code behind build tags or environment checks
- Use structured logging for better debugging experience
- Implement proper error handling and validation

### 3. Testing Strategy

- Test functions individually with `sam local invoke`
- Test API integration with `sam local start-api`
- Use real AWS services for integration testing
- Create comprehensive test events for different scenarios

### 4. Performance Considerations

- Local execution is slower than AWS Lambda
- Docker container startup adds overhead
- Use `--warm-containers EAGER` for faster subsequent invocations

### 5. Security

- Don't commit AWS credentials to version control
- Use environment variables or AWS credential files
- Test with minimal IAM permissions locally

## Advanced Usage

### Multiple Function Debugging

To debug multiple functions, you need separate debug sessions:

```bash
# Terminal 1: Debug hello function
sam local start-api --debug-port 5986 --debug-function aws_lambda_function.hello --hook-name terraform --beta-features

# Terminal 2: Debug users function (different port)
sam local start-api --debug-port 5987 --debug-function aws_lambda_function.users --hook-name terraform --beta-features
```

### Custom Docker Networks

For testing with local databases or services:

```bash
# Create Docker network
docker network create sam-local

# Start local services on the network
docker run --network sam-local --name postgres -e POSTGRES_PASSWORD=password -d postgres

# Use network with SAM
sam local start-api --docker-network sam-local --hook-name terraform --beta-features
```

### Testing with Real AWS Resources

SAM local can interact with real AWS services:

```bash
# Set AWS region and profile
export AWS_REGION=us-east-1
export AWS_PROFILE=your-profile

# Invoke with real AWS services
sam local invoke --hook-name terraform --beta-features aws_lambda_function.users -e ../events/users-event.json
```

This setup allows you to test locally while reading/writing to actual DynamoDB tables, S3 buckets, etc.

## Conclusion

SAM CLI with Terraform provides a powerful local development environment for Go Lambda functions. This setup enables:

- Fast development iteration
- Comprehensive debugging capabilities
- Integration testing with real AWS services
- Cost-effective local testing

The combination of SAM's local execution environment and Go's excellent debugging tools creates an efficient development workflow for serverless applications.

For more information, refer to:
- [AWS SAM CLI Documentation](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-command-reference.html)
- [AWS SAM + Terraform Blog Post](https://aws.amazon.com/blogs/compute/better-together-aws-sam-cli-and-hashicorp-terraform/)
- [Delve Debugger Documentation](https://github.com/go-delve/delve)
