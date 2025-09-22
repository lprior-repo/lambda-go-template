package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Request represents the input to the docs generator
type Request struct {
	Action  string `json:"action"`
	SpecKey string `json:"spec_key"`
}

// Response represents the output from the docs generator
type Response struct {
	StatusCode int               `json:"statusCode"`
	Body       string            `json:"body"`
	Headers    map[string]string `json:"headers"`
}

// DocsGenerator handles documentation generation
type DocsGenerator struct {
	s3Client   *s3.S3
	bucketName string
}

// NewDocsGenerator creates a new docs generator
func NewDocsGenerator() *DocsGenerator {
	sess := session.Must(session.NewSession())

	return &DocsGenerator{
		s3Client:   s3.New(sess),
		bucketName: os.Getenv("DOCS_BUCKET"),
	}
}

// generateRedocHTML generates a Redoc HTML page
func (dg *DocsGenerator) generateRedocHTML(specURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>Lambda Go Template API Documentation</title>
	<meta charset="utf-8"/>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
	<style>
		body {
			margin: 0;
			padding: 0;
		}
		.header {
			background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
			color: white;
			padding: 2rem 0;
			text-align: center;
			box-shadow: 0 2px 10px rgba(0,0,0,0.1);
		}
		.header h1 {
			margin: 0;
			font-family: 'Montserrat', sans-serif;
			font-weight: 300;
			font-size: 2.5rem;
		}
		.header p {
			margin: 1rem 0 0 0;
			font-family: 'Roboto', sans-serif;
			font-size: 1.1rem;
			opacity: 0.9;
		}
		.badges {
			margin-top: 1rem;
		}
		.badge {
			display: inline-block;
			background: rgba(255,255,255,0.2);
			padding: 0.3rem 0.8rem;
			border-radius: 15px;
			margin: 0 0.2rem;
			font-size: 0.8rem;
			font-weight: 500;
		}
		#redoc-container {
			margin-top: 0;
		}
	</style>
</head>
<body>
	<div class="header">
		<h1>Lambda Go Template API</h1>
		<p>Production-ready Go Lambda microservice with observability</p>
		<div class="badges">
			<span class="badge">Go 1.21</span>
			<span class="badge">AWS Lambda</span>
			<span class="badge">OpenAPI 3.0</span>
			<span class="badge">ARM64</span>
		</div>
	</div>
	<div id="redoc-container"></div>
	<script src="https://cdn.redoc.ly/redoc/2.1.3/bundles/redoc.standalone.js"></script>
	<script>
		Redoc.init('%s', {
			scrollYOffset: 50,
			theme: {
				colors: {
					primary: {
						main: '#667eea'
					}
				},
				typography: {
					fontSize: '14px',
					lineHeight: '1.5em',
					code: {
						fontSize: '13px',
						fontFamily: 'Courier, monospace'
					},
					headings: {
						fontFamily: 'Montserrat, sans-serif',
						fontWeight: '400'
					}
				},
				sidebar: {
					width: '260px'
				}
			},
			hideDownloadButton: false,
			expandResponses: "200,201",
			jsonSampleExpandLevel: 2,
			hideSingleRequestSampleTab: true,
			menuToggle: true,
			sortPropsAlphabetically: true,
			payloadSampleIdx: 0
		}, document.getElementById('redoc-container'));
	</script>
</body>
</html>`, specURL)
}

// generatePostmanCollection generates a Postman collection from OpenAPI spec
func (dg *DocsGenerator) generatePostmanCollection(apiGatewayURL string) string {
	collection := map[string]interface{}{
		"info": map[string]interface{}{
			"name":        "Lambda Go Template API",
			"description": "Production-ready Go Lambda microservice API",
			"schema":      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		"variable": []map[string]interface{}{
			{
				"key":   "baseUrl",
				"value": apiGatewayURL,
				"type":  "string",
			},
		},
		"item": []map[string]interface{}{
			{
				"name": "Health Check",
				"request": map[string]interface{}{
					"method": "GET",
					"header": []map[string]interface{}{
						{
							"key":   "Content-Type",
							"value": "application/json",
						},
					},
					"url": map[string]interface{}{
						"raw":  "{{baseUrl}}/prod/hello",
						"host": []string{"{{baseUrl}}"},
						"path": []string{"prod", "hello"},
					},
					"description": "Health check endpoint with service information",
				},
				"response": []interface{}{},
			},
			{
				"name": "List Users",
				"request": map[string]interface{}{
					"method": "GET",
					"header": []map[string]interface{}{
						{
							"key":   "Content-Type",
							"value": "application/json",
						},
					},
					"url": map[string]interface{}{
						"raw":   "{{baseUrl}}/prod/users",
						"host":  []string{"{{baseUrl}}"},
						"path":  []string{"prod", "users"},
						"query": []map[string]interface{}{
							{
								"key":   "limit",
								"value": "10",
								"description": "Maximum number of users to return",
								"disabled": true,
							},
							{
								"key":   "offset",
								"value": "0",
								"description": "Number of users to skip",
								"disabled": true,
							},
						},
					},
					"description": "Retrieve a list of all users",
				},
				"response": []interface{}{},
			},
		},
	}

	jsonBytes, _ := json.MarshalIndent(collection, "", "  ")
	return string(jsonBytes)
}

// generateReadme generates a README file for the API
func (dg *DocsGenerator) generateReadme(apiGatewayURL string) string {
	readme := "# Lambda Go Template API\n\n"
	readme += "Production-ready Go Lambda microservice with observability and best practices.\n\n"
	readme += "## 🚀 Quick Start\n\n"
	readme += "### Base URL\n"
	readme += apiGatewayURL + "\n\n"
	readme += "### Available Endpoints\n\n"
	readme += "#### Health Check\n"
	readme += "- **GET** `/prod/hello` - Service health and information\n\n"
	readme += "#### Users\n"
	readme += "- **GET** `/prod/users` - List all users\n"
	readme += "- **GET** `/prod/users/{id}` - Get user by ID (coming soon)\n\n"
	readme += "### Example Requests\n\n"
	readme += "#### Health Check\n"
	readme += "```bash\n"
	readme += "curl -X GET \"" + apiGatewayURL + "/prod/hello\"\n"
	readme += "```\n\n"
	readme += "#### List Users\n"
	readme += "```bash\n"
	readme += "curl -X GET \"" + apiGatewayURL + "/prod/users\"\n"
	readme += "```\n\n"
	readme += "## 📊 Features\n\n"
	readme += "- ✅ **Idiomatic Go** - Clean, maintainable Go code\n"
	readme += "- ✅ **Observability** - Structured logging & X-Ray tracing\n"
	readme += "- ✅ **Error Handling** - Comprehensive error responses\n"
	readme += "- ✅ **Security** - CORS, security headers, validation\n"
	readme += "- ✅ **Performance** - ARM64 optimized Lambda functions\n"
	readme += "- ✅ **Infrastructure as Code** - Complete Terraform setup\n"
	readme += "- ✅ **Testing** - Unit, integration & performance tests\n\n"
	readme += "Generated on " + time.Now().Format("2006-01-02 15:04:05 UTC") + "\n"

	return readme
}

// uploadToS3 uploads content to S3
func (dg *DocsGenerator) uploadToS3(key, content, contentType string) error {
	_, err := dg.s3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(dg.bucketName),
		Key:         aws.String(key),
		Body:        strings.NewReader(content),
		ContentType: aws.String(contentType),
		CacheControl: aws.String("max-age=3600"),
	})
	return err
}

// generateDocs generates all documentation files
func (dg *DocsGenerator) generateDocs(apiGatewayURL string) error {
	specURL := fmt.Sprintf("https://%s.s3.amazonaws.com/openapi.yaml", dg.bucketName)

	// Generate Redoc HTML
	redocHTML := dg.generateRedocHTML(specURL)
	if err := dg.uploadToS3("redoc.html", redocHTML, "text/html"); err != nil {
		return fmt.Errorf("failed to upload redoc.html: %v", err)
	}

	// Generate Postman collection
	postmanCollection := dg.generatePostmanCollection(apiGatewayURL)
	if err := dg.uploadToS3("postman-collection.json", postmanCollection, "application/json"); err != nil {
		return fmt.Errorf("failed to upload postman collection: %v", err)
	}

	// Generate README
	readme := dg.generateReadme(apiGatewayURL)
	if err := dg.uploadToS3("README.md", readme, "text/markdown"); err != nil {
		return fmt.Errorf("failed to upload README.md: %v", err)
	}

	// Generate a simple API status page
	statusPage := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>API Status - Lambda Go Template</title>
	<meta charset="utf-8"/>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
		.container { max-width: 800px; margin: 0 auto; background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
		.status { color: #4caf50; font-weight: bold; }
		.endpoint { background: #f8f9fa; padding: 15px; margin: 10px 0; border-radius: 5px; border-left: 4px solid #4caf50; }
		.endpoint h3 { margin: 0 0 10px 0; color: #333; }
		.endpoint code { background: #e9ecef; padding: 2px 6px; border-radius: 3px; }
		a { color: #007bff; text-decoration: none; }
		a:hover { text-decoration: underline; }
	</style>
</head>
<body>
	<div class="container">
		<h1>Lambda Go Template API</h1>
		<p class="status">🟢 API Status: Operational</p>
		<p>Last updated: %s</p>

		<h2>Available Endpoints</h2>
		<div class="endpoint">
			<h3>Health Check</h3>
			<p><strong>GET</strong> <code>%s/prod/hello</code></p>
			<p>Returns service health and version information</p>
		</div>

		<div class="endpoint">
			<h3>Users</h3>
			<p><strong>GET</strong> <code>%s/prod/users</code></p>
			<p>Returns a list of users with metadata</p>
		</div>

		<h2>Documentation</h2>
		<ul>
			<li><a href="index.html">Interactive Swagger UI</a></li>
			<li><a href="redoc.html">Redoc Documentation</a></li>
			<li><a href="openapi.yaml">OpenAPI Specification (YAML)</a></li>
			<li><a href="openapi.json">OpenAPI Specification (JSON)</a></li>
			<li><a href="postman-collection.json">Postman Collection</a></li>
			<li><a href="README.md">API README</a></li>
		</ul>
	</div>
</body>
</html>`, time.Now().Format("2006-01-02 15:04:05 UTC"), apiGatewayURL, apiGatewayURL)

	if err := dg.uploadToS3("status.html", statusPage, "text/html"); err != nil {
		return fmt.Errorf("failed to upload status.html: %v", err)
	}

	return nil
}

// Handler handles Lambda invocations
func (dg *DocsGenerator) Handler(ctx context.Context, event json.RawMessage) (Response, error) {
	log.Printf("Received event: %s", string(event))

	// Try to parse as direct invocation first
	var request Request
	if err := json.Unmarshal(event, &request); err != nil {
		// Try to parse as API Gateway event
		var apiEvent events.APIGatewayProxyRequest
		if err := json.Unmarshal(event, &apiEvent); err != nil {
			log.Printf("Failed to parse event: %v", err)
			return Response{
				StatusCode: 400,
				Body:       `{"error": "Invalid event format"}`,
				Headers:    map[string]string{"Content-Type": "application/json"},
			}, nil
		}

		// Handle API Gateway request
		return Response{
			StatusCode: 200,
			Body:       `{"message": "Documentation generator is running", "status": "ok"}`,
			Headers: map[string]string{
				"Content-Type": "application/json",
				"Access-Control-Allow-Origin": "*",
			},
		}, nil
	}

	// Handle direct invocation
	if request.Action == "generate" {
		apiGatewayURL := os.Getenv("API_GATEWAY_URL")
		if apiGatewayURL == "" {
			apiGatewayURL = "https://api.example.com"
		}

		log.Printf("Generating documentation for API: %s", apiGatewayURL)

		if err := dg.generateDocs(apiGatewayURL); err != nil {
			log.Printf("Failed to generate docs: %v", err)
			return Response{
				StatusCode: 500,
				Body:       fmt.Sprintf(`{"error": "Failed to generate documentation: %v"}`, err),
				Headers:    map[string]string{"Content-Type": "application/json"},
			}, nil
		}

		log.Println("Documentation generated successfully")
		return Response{
			StatusCode: 200,
			Body:       `{"message": "Documentation generated successfully", "status": "ok"}`,
			Headers:    map[string]string{"Content-Type": "application/json"},
		}, nil
	}

	return Response{
		StatusCode: 400,
		Body:       `{"error": "Unknown action"}`,
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

func main() {
	dg := NewDocsGenerator()
	lambda.Start(dg.Handler)
}
