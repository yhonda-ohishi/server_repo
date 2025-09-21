package gateway

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
)

// Using CDN for Swagger UI instead of embedding files

// SwaggerSpec represents the OpenAPI specification
type SwaggerSpec struct {
	OpenAPI string                 `json:"openapi"`
	Info    SwaggerInfo            `json:"info"`
	Servers []SwaggerServer        `json:"servers"`
	Paths   map[string]interface{} `json:"paths"`
}

type SwaggerInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type SwaggerServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

// SetupSwaggerUI configures Swagger UI routes
func (g *SimpleGateway) SetupSwaggerUI() {
	// Serve auto-generated swagger spec JSON from protobuf
	g.app.Get("/swagger.json", func(c *fiber.Ctx) error {
		// Try to read db_service swagger file
		dbServiceSwaggerPath := filepath.Join("..", "db_service", "swagger", "apidocs.swagger.json")
		var dbServiceSwagger map[string]interface{}

		if data, err := ioutil.ReadFile(dbServiceSwaggerPath); err == nil {
			if err := json.Unmarshal(data, &dbServiceSwagger); err == nil {
				log.Printf("Successfully loaded db_service swagger")
				// Return db_service swagger directly
				return c.JSON(dbServiceSwagger)
			}
		}

		// Try to read the auto-generated swagger file first
		swaggerPath := filepath.Join("swagger", "etc_service.swagger.json")
		if data, err := ioutil.ReadFile(swaggerPath); err == nil {
			var swaggerData interface{}
			if err := json.Unmarshal(data, &swaggerData); err == nil {
				return c.JSON(swaggerData)
			}
		}

		// Fallback to manual swagger spec if auto-generated file is not available
		log.Printf("Auto-generated swagger file not found, falling back to manual spec")
		spec := g.generateSwaggerSpec()
		return c.JSON(spec)
	})

	// Serve Swagger UI
	g.app.Get("/docs", func(c *fiber.Ctx) error {
		html := g.generateSwaggerHTML()
		c.Set("Content-Type", "text/html")
		return c.SendString(html)
	})

	// API documentation route
	g.app.Get("/api-docs", func(c *fiber.Ctx) error {
		return c.Redirect("/docs")
	})

	// Health check for swagger
	g.app.Get("/swagger/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"swagger": "available",
			"endpoints": []string{
				"/docs",
				"/swagger.json",
				"/api-docs",
			},
		})
	})
}

// generateSwaggerSpec creates OpenAPI 3.0 specification
func (g *SimpleGateway) generateSwaggerSpec() *SwaggerSpec {
	return &SwaggerSpec{
		OpenAPI: "3.0.0",
		Info: SwaggerInfo{
			Title:       "gRPC-First Multi-Protocol Gateway API",
			Description: "API documentation for ETC Meisai Gateway supporting REST, gRPC, and JSON-RPC protocols",
			Version:     "1.0.0",
		},
		Servers: []SwaggerServer{
			{
				URL:         "http://localhost:8081",
				Description: "Development server",
			},
		},
		Paths: map[string]interface{}{
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Health check",
					"description": "Returns the health status of the gateway",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Healthy",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status": map[string]interface{}{
												"type": "string",
											},
											"timestamp": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/users": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List users",
					"description": "Retrieve a list of users with pagination",
					"parameters": []map[string]interface{}{
						{
							"name":        "page_size",
							"in":          "query",
							"description": "Number of users to return",
							"schema": map[string]interface{}{
								"type":    "integer",
								"default": 10,
							},
						},
						{
							"name":        "page_token",
							"in":          "query",
							"description": "Token for pagination",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of users",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"users": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"$ref": "#/components/schemas/User",
												},
											},
											"next_page_token": map[string]interface{}{
												"type": "string",
											},
										},
									},
								},
							},
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Create user",
					"description": "Create a new user",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/CreateUserRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "User created",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/User",
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/users/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get user by ID",
					"parameters": []map[string]interface{}{
						{
							"name":     "id",
							"in":       "path",
							"required": true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "User details",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/User",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "User not found",
						},
					},
				},
			},
			"/api/v1/transactions/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get transaction by ID",
					"parameters": []map[string]interface{}{
						{
							"name":     "id",
							"in":       "path",
							"required": true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Transaction details",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Transaction",
									},
								},
							},
						},
					},
				},
			},
			"/jsonrpc": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "JSON-RPC 2.0 endpoint",
					"description": "Execute JSON-RPC 2.0 methods",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/JsonRpcRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "JSON-RPC response",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/JsonRpcResponse",
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/etc/meisai": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"ETC明細"},
					"summary":     "ETC明細一覧取得",
					"description": "ETC明細データの一覧を取得します",
					"parameters": []map[string]interface{}{
						{
							"name":        "start_date",
							"in":          "query",
							"description": "開始日 (YYYY-MM-DD)",
							"schema": map[string]interface{}{
								"type": "string",
								"format": "date",
							},
						},
						{
							"name":        "end_date",
							"in":          "query",
							"description": "終了日 (YYYY-MM-DD)",
							"schema": map[string]interface{}{
								"type": "string",
								"format": "date",
							},
						},
						{
							"name":        "page_size",
							"in":          "query",
							"description": "1ページあたりの件数",
							"schema": map[string]interface{}{
								"type": "integer",
								"default": 10,
								"minimum": 1,
								"maximum": 100,
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "ETC明細一覧",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ETCMeisaiListResponse",
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/etc/summary": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"ETC明細"},
					"summary":     "ETC利用サマリー取得",
					"description": "ETC明細データのサマリー情報を取得します",
					"parameters": []map[string]interface{}{
						{
							"name":        "start_date",
							"in":          "query",
							"description": "集計開始日 (YYYY-MM-DD)",
							"schema": map[string]interface{}{
								"type": "string",
								"format": "date",
							},
						},
						{
							"name":        "end_date",
							"in":          "query",
							"description": "集計終了日 (YYYY-MM-DD)",
							"schema": map[string]interface{}{
								"type": "string",
								"format": "date",
							},
						},
						{
							"name":        "user_id",
							"in":          "query",
							"description": "ユーザーID（指定時は該当ユーザーのみ集計）",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "ETC利用サマリー",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ETCSummaryResponse",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// generateSwaggerHTML generates the Swagger UI HTML page
func (g *SimpleGateway) generateSwaggerHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>gRPC-First Multi-Protocol Gateway API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        *, *:before, *:after {
            box-sizing: inherit;
        }
        body {
            margin:0;
            background: #fafafa;
        }
        .swagger-ui .topbar {
            background-color: #2c3e50;
        }
        .swagger-ui .topbar .download-url-wrapper {
            display: none;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/swagger.json',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                tryItOutEnabled: true,
                requestInterceptor: function(request) {
                    request.headers['Content-Type'] = 'application/json';
                    return request;
                },
                responseInterceptor: function(response) {
                    return response;
                }
            });
        };
    </script>
</body>
</html>`
}

// AddSwaggerSchemas adds component schemas to swagger spec
func (spec *SwaggerSpec) AddSwaggerSchemas() {
	if spec.Paths == nil {
		spec.Paths = make(map[string]interface{})
	}

	// Add components section with schemas
	components := map[string]interface{}{
		"schemas": map[string]interface{}{
			"User": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type": "string",
					},
					"email": map[string]interface{}{
						"type": "string",
						"format": "email",
					},
					"name": map[string]interface{}{
						"type": "string",
					},
					"phone_number": map[string]interface{}{
						"type": "string",
					},
					"address": map[string]interface{}{
						"type": "string",
					},
					"status": map[string]interface{}{
						"type": "string",
						"enum": []string{"active", "inactive"},
					},
					"created_at": map[string]interface{}{
						"type": "string",
						"format": "date-time",
					},
					"updated_at": map[string]interface{}{
						"type": "string",
						"format": "date-time",
					},
				},
			},
			"CreateUserRequest": map[string]interface{}{
				"type": "object",
				"required": []string{"email", "name"},
				"properties": map[string]interface{}{
					"email": map[string]interface{}{
						"type": "string",
						"format": "email",
					},
					"name": map[string]interface{}{
						"type": "string",
					},
					"phone_number": map[string]interface{}{
						"type": "string",
					},
					"address": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"Transaction": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type": "string",
					},
					"card_id": map[string]interface{}{
						"type": "string",
					},
					"entry_gate_id": map[string]interface{}{
						"type": "string",
					},
					"exit_gate_id": map[string]interface{}{
						"type": "string",
					},
					"entry_time": map[string]interface{}{
						"type": "string",
						"format": "date-time",
					},
					"exit_time": map[string]interface{}{
						"type": "string",
						"format": "date-time",
					},
					"distance": map[string]interface{}{
						"type": "number",
						"format": "float",
					},
					"toll_amount": map[string]interface{}{
						"type": "integer",
					},
					"discount_amount": map[string]interface{}{
						"type": "integer",
					},
					"final_amount": map[string]interface{}{
						"type": "integer",
					},
					"payment_status": map[string]interface{}{
						"type": "string",
						"enum": []string{"pending", "completed", "failed"},
					},
					"transaction_date": map[string]interface{}{
						"type": "string",
						"format": "date-time",
					},
				},
			},
			"JsonRpcRequest": map[string]interface{}{
				"type": "object",
				"required": []string{"jsonrpc", "method", "id"},
				"properties": map[string]interface{}{
					"jsonrpc": map[string]interface{}{
						"type": "string",
						"enum": []string{"2.0"},
					},
					"method": map[string]interface{}{
						"type": "string",
					},
					"params": map[string]interface{}{
						"type": "object",
					},
					"id": map[string]interface{}{
						"oneOf": []map[string]interface{}{
							{"type": "string"},
							{"type": "integer"},
							{"type": "null"},
						},
					},
				},
			},
			"JsonRpcResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"jsonrpc": map[string]interface{}{
						"type": "string",
						"enum": []string{"2.0"},
					},
					"result": map[string]interface{}{
						"type": "object",
					},
					"error": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"code": map[string]interface{}{
								"type": "integer",
							},
							"message": map[string]interface{}{
								"type": "string",
							},
							"data": map[string]interface{}{
								"type": "object",
							},
						},
					},
					"id": map[string]interface{}{
						"oneOf": []map[string]interface{}{
							{"type": "string"},
							{"type": "integer"},
							{"type": "null"},
						},
					},
				},
			},
		},
	}

	// Convert spec to map and add components
	specMap := make(map[string]interface{})
	specBytes, _ := json.Marshal(spec)
	json.Unmarshal(specBytes, &specMap)
	specMap["components"] = components

	// Update spec
	updatedBytes, _ := json.Marshal(specMap)
	json.Unmarshal(updatedBytes, spec)
}