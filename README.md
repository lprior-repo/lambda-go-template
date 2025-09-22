# Lambda Go Template

This template provides a complete setup for AWS Lambda functions written in Go, using Terraform for infrastructure management.

## Structure

```
.
├── src/
│   ├── hello/          # Hello Lambda function
│   │   └── main.go
│   └── users/          # Users Lambda function
│       └── main.go
├── terraform/          # Terraform infrastructure
│   ├── main.tf
│   ├── variables.tf
│   └── outputs.tf
├── .github/
│   └── workflows/
│       └── build.yml   # GitHub Actions CI/CD
├── events/             # SAM test events
│   ├── hello-event.json
│   └── users-*-event.json
├── docs/               # Documentation
│   ├── SAM_LOCAL_DEBUGGING.md
│   └── SAM_QUICK_START.md
├── go.mod
├── Taskfile.yml        # Task automation
├── samconfig.toml      # SAM CLI configuration
└── README.md
```

## Prerequisites

- Go 1.21+
- Terraform >= 1.13
- AWS CLI configured
- Task (optional, for using Taskfile commands)
- AWS SAM CLI (for local debugging)
- Docker (required for SAM local execution)

## Getting Started

1. **Clone this template**
   ```bash
   git clone <this-repo>
   cd lambda-go-template
   ```

2. **Install dependencies**
   ```bash
   go mod download
   # or
   make deps
   ```

3. **Build Lambda functions**
   ```bash
   make build
   ```

4. **Deploy infrastructure**
   ```bash
   cd terraform
   terraform init
   terraform plan
   terraform apply
   # or
   make deploy
   ```

## Development

### Local Development with SAM

For local testing and debugging, this project supports AWS SAM CLI integration:

```bash
# Quick start - build and test locally
task sam:build          # Build SAM application
task sam:invoke:hello   # Test hello function
task sam:api            # Start local API Gateway

# Debugging
task debug:hello        # Debug hello function (attach on port 5986)
task debug:api          # Debug with local API Gateway
```

See [SAM Quick Start Guide](docs/SAM_QUICK_START.md) for a 5-minute setup, or [SAM Local Debugging Guide](docs/SAM_LOCAL_DEBUGGING.md) for comprehensive documentation.

### Adding a New Function

1. Create a new directory under `src/` (e.g., `src/orders/`)
2. Add your `main.go` file
3. Add the function to `terraform/main.tf`
4. The build process will automatically detect and build the new function

### Building

```bash
# Build all functions (production)
task build

# Build for local debugging (with debug symbols)
task build:debug

# Build and package for deployment
task package
```

### Testing

```bash
# Run all tests with coverage
task test

# Run tests in watch mode (TDD)
task test:watch

# Test locally with SAM
task sam:test

# Test individual functions
task sam:invoke:hello
task sam:invoke:users
```

### Linting

```bash
# Run all linting and validation
task validate

# Run linting only
task lint

# Install development tools
task dev:tools
```

## CI/CD

The GitHub Actions workflow automatically:
- Detects changed functions
- Builds each function in parallel
- Runs tests and linting
- Creates deployment packages
- Uploads build artifacts

## Terraform Configuration

The infrastructure uses:
- **terraform-aws-modules/lambda/aws** for Lambda functions
- **terraform-aws-modules/apigateway-v2/aws** for API Gateway
- Pre-built packages (no building in Terraform)

### Customization

Edit `terraform/variables.tf` to customize:
- AWS region
- Function names
- Environment settings

## API Endpoints

After deployment, you'll get:
- `GET /hello` - Hello function
- `GET /users` - Users function

## Local Development & Debugging

This project includes comprehensive SAM CLI integration for local development:

### Quick Commands
```bash
# Start local API Gateway
task sam:api

# Debug functions with breakpoints
task debug:hello        # VS Code: F5 to attach debugger
task debug:users

# Test individual functions
task sam:invoke:hello
task sam:invoke:users
```

### IDE Integration
- **VS Code**: Configured debug settings in `.vscode/launch.json`
- **Command Line**: Uses Delve debugger (`dlv connect localhost:5986`)
- **Test Events**: Pre-configured events in `events/` directory

See [SAM Quick Start](docs/SAM_QUICK_START.md) for immediate setup or [SAM Debugging Guide](docs/SAM_LOCAL_DEBUGGING.md) for detailed documentation.

## Cost Optimization

- Functions use `provided.al2023` runtime (most cost-effective for Go)
- CloudWatch logs have 14-day retention
- API Gateway uses HTTP API (cheaper than REST API)

## Security

- IAM roles follow least privilege principle
- CloudWatch logs enabled for monitoring
- Gosec security scanning in CI/CD
