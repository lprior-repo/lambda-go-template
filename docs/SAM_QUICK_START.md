# SAM Local Debugging - Quick Start Guide

This is a quick start guide to get you up and running with SAM local debugging for the lambda-go-template project.

## Prerequisites Check

Make sure you have these installed:

```bash
# Check installations
sam --version          # Should show SAM CLI version
go version            # Should show Go 1.21+
docker --version      # Should show Docker version
dlv version          # Should show Delve debugger version
```

If missing any, install them:

```bash
# Install SAM CLI (macOS)
brew install aws-sam-cli

# Install Delve debugger
go install github.com/go-delve/delve/cmd/dlv@latest
```

## Quick Setup (5 minutes)

### 1. Build Lambda Functions for Debugging

```bash
# From project root
cd src/hello
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -gcflags="all=-N -l" -o bootstrap main.go
zip -r ../../build/hello.zip bootstrap
rm bootstrap

cd ../users
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -gcflags="all=-N -l" -o bootstrap main.go
zip -r ../../build/users.zip bootstrap
rm bootstrap

cd ../..
```

### 2. Build SAM Application

```bash
cd terraform
sam build --hook-name terraform --beta-features
```

### 3. Test Local Invocation

```bash
# Test hello function
sam local invoke aws_lambda_function.hello -e ../events/hello-event.json --beta-features

# Test users function
sam local invoke aws_lambda_function.users -e ../events/users-get-event.json --beta-features
```

### 4. Start Local API

```bash
# Start local API Gateway
sam local start-api --hook-name terraform --beta-features --port 3000

# Test in another terminal
curl http://localhost:3000/hello
curl http://localhost:3000/users
```

## Quick Debugging (3 steps)

### Option A: VS Code Debugging

1. **Start function in debug mode:**
   ```bash
   cd terraform
   sam local invoke aws_lambda_function.hello -e ../events/hello-event.json --debug-port 5986 --beta-features
   ```

2. **Set breakpoints** in your Go code (e.g., `src/hello/main.go`)

3. **Attach debugger** in VS Code:
   - Go to Run and Debug panel (Ctrl+Shift+D)
   - Select "Debug Lambda Hello"
   - Click Start Debugging (F5)

### Option B: Command Line Debugging

1. **Start function in debug mode:**
   ```bash
   cd terraform
   sam local invoke aws_lambda_function.hello -e ../events/hello-event.json --debug-port 5986 --beta-features
   ```

2. **Connect with Delve** (in another terminal):
   ```bash
   dlv connect localhost:5986
   ```

3. **Set breakpoints and debug:**
   ```
   (dlv) break main.main
   (dlv) continue
   (dlv) step
   (dlv) print variableName
   ```

## Common Commands

```bash
# Quick development cycle
make build                                           # Build functions
cd terraform && sam build --hook-name terraform --beta-features  # Build SAM
sam local start-api --hook-name terraform --beta-features        # Start API

# Debug specific function
sam local invoke aws_lambda_function.hello -e ../events/hello-event.json --debug-port 5986 --beta-features

# Debug with API Gateway
sam local start-api --hook-name terraform --beta-features --debug-port 5986

# Use different environments
sam local invoke --config-env debug aws_lambda_function.hello -e ../events/hello-event.json
```

## Test Events

Pre-created test events in `events/` directory:

- `hello-event.json` - GET /hello request
- `users-get-event.json` - GET /users request
- `users-post-event.json` - POST /users request

Usage:
```bash
sam local invoke aws_lambda_function.hello -e ../events/hello-event.json --beta-features
sam local invoke aws_lambda_function.users -e ../events/users-post-event.json --beta-features
```

## Environment Configurations

The project includes environment-specific configurations:

- `env-dev.json` - Development environment variables
- `env-debug.json` - Debug environment variables

Use with:
```bash
sam local invoke --env-vars ../env-debug.json aws_lambda_function.hello -e ../events/hello-event.json --beta-features
```

## Troubleshooting

### Function not found
```bash
# Make sure you're in terraform/ directory
cd terraform
sam build --hook-name terraform --beta-features
```

### Docker issues
```bash
# Check Docker is running
docker ps

# Pull latest images
sam local invoke --force-image-build --hook-name terraform --beta-features aws_lambda_function.hello
```

### Debugger won't connect
```bash
# Check if port is available
lsof -i :5986

# Try different port
sam local invoke --debug-port 5987 --hook-name terraform --beta-features aws_lambda_function.hello
```

### Build issues
```bash
# Clean build
rm -rf build/*
make build

# Verify zip contents
unzip -l build/hello.zip
```

## Next Steps

- Read the full [SAM_LOCAL_DEBUGGING.md](./SAM_LOCAL_DEBUGGING.md) for comprehensive documentation
- Explore advanced debugging features
- Set up continuous testing with SAM
- Configure your IDE for optimal debugging experience

## Useful Links

- [AWS SAM CLI Documentation](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-command-reference.html)
- [Delve Debugger Commands](https://github.com/go-delve/delve/tree/master/Documentation/cli)
- [VS Code Go Debugging](https://code.visualstudio.com/docs/languages/go#_debugging)
