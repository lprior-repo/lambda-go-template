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
├── go.mod
├── Makefile
└── README.md
```

## Prerequisites

- Go 1.21+
- Terraform >= 1.0
- AWS CLI configured
- Make (optional, for using Makefile commands)

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

### Adding a New Function

1. Create a new directory under `src/` (e.g., `src/orders/`)
2. Add your `main.go` file
3. Add the function to `terraform/main.tf`
4. The build process will automatically detect and build the new function

### Building

```bash
# Build all functions
make build

# Or manually
mkdir -p build
for func in $(find src -mindepth 1 -maxdepth 1 -type d -exec basename {} \;); do
    cd src/$func
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bootstrap main.go
    zip -r ../../build/$func.zip bootstrap
    rm bootstrap
    cd ../..
done
```

### Testing

```bash
# Run tests
make test

# Or manually
go test ./...
```

### Linting

```bash
# Run linter
make lint

# Install golangci-lint first
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
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

## Cost Optimization

- Functions use `provided.al2023` runtime (most cost-effective for Go)
- CloudWatch logs have 14-day retention
- API Gateway uses HTTP API (cheaper than REST API)

## Security

- IAM roles follow least privilege principle
- CloudWatch logs enabled for monitoring
- Gosec security scanning in CI/CD