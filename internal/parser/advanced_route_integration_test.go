package parser

import (
	"strings"
	"testing"

	"github.com/toyz/axon/internal/models"
)

func TestAdvancedRouteFeatures_Integration(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected models.PackageMetadata
	}{
		{
			name: "route with echo syntax parameters",
			source: `package test

import "github.com/labstack/echo/v4"

//axon::controller
type UserController struct{}

//axon::route GET /users/:id/posts/:slug
func (uc *UserController) GetUserPost(id string, slug string) (interface{}, error) {
	return nil, nil
}`,
			expected: models.PackageMetadata{
				PackageName: "test",
				Controllers: []models.ControllerMetadata{
					{
						Name:       "UserController",
						StructName: "UserController",
						Routes: []models.RouteMetadata{
							{
								Method:      "GET",
								Path:        "/users/:id/posts/:slug",
								HandlerName: "GetUserPost",
								Parameters: []models.Parameter{
									{
										Name:     "id",
										Type:     "string",
										Source:   models.ParameterSourcePath,
										Required: true,
									},
									{
										Name:     "slug",
										Type:     "string",
										Source:   models.ParameterSourcePath,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "route with mixed parameter syntax",
			source: `package test

import "github.com/labstack/echo/v4"

//axon::controller
type UserController struct{}

//axon::route GET /users/{id:int}/posts/:slug
func (uc *UserController) GetUserPost(id int, slug string) (interface{}, error) {
	return nil, nil
}`,
			expected: models.PackageMetadata{
				PackageName: "test",
				Controllers: []models.ControllerMetadata{
					{
						Name:       "UserController",
						StructName: "UserController",
						Routes: []models.RouteMetadata{
							{
								Method:      "GET",
								Path:        "/users/{id:int}/posts/:slug",
								HandlerName: "GetUserPost",
								Parameters: []models.Parameter{
									{
										Name:     "id",
										Type:     "int",
										Source:   models.ParameterSourcePath,
										Required: true,
									},
									{
										Name:     "slug",
										Type:     "string",
										Source:   models.ParameterSourcePath,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "route with context parameter detection",
			source: `package test

import "github.com/labstack/echo/v4"

//axon::controller
type UserController struct{}

//axon::route GET /users/{id:int}
func (uc *UserController) GetUser(c echo.Context, id int) (interface{}, error) {
	return nil, nil
}`,
			expected: models.PackageMetadata{
				PackageName: "test",
				Controllers: []models.ControllerMetadata{
					{
						Name:       "UserController",
						StructName: "UserController",
						Routes: []models.RouteMetadata{
							{
								Method:      "GET",
								Path:        "/users/{id:int}",
								HandlerName: "GetUser",
								Parameters: []models.Parameter{
									{
										Name:     "id",
										Type:     "int",
										Source:   models.ParameterSourcePath,
										Required: true,
									},
									{
										Name:     "c",
										Type:     "echo.Context",
										Source:   models.ParameterSourceContext,
										Required: true,
										Position: 0,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "route with PassContext flag",
			source: `package test

import "github.com/labstack/echo/v4"

//axon::controller
type UserController struct{}

//axon::route GET /health -PassContext
func (uc *UserController) HealthCheck() error {
	return nil
}`,
			expected: models.PackageMetadata{
				PackageName: "test",
				Controllers: []models.ControllerMetadata{
					{
						Name:       "UserController",
						StructName: "UserController",
						Routes: []models.RouteMetadata{
							{
								Method:      "GET",
								Path:        "/health",
								HandlerName: "HealthCheck",
								Flags:       []string{"-PassContext"},
								Parameters:  []models.Parameter{},
							},
						},
					},
				},
			},
		},
		{
			name: "route with context parameter in middle position",
			source: `package test

import "github.com/labstack/echo/v4"

//axon::controller
type UserController struct{}

//axon::route PUT /users/:id
func (uc *UserController) UpdateUser(id string, ctx echo.Context, data string) error {
	return nil
}`,
			expected: models.PackageMetadata{
				PackageName: "test",
				Controllers: []models.ControllerMetadata{
					{
						Name:       "UserController",
						StructName: "UserController",
						Routes: []models.RouteMetadata{
							{
								Method:      "PUT",
								Path:        "/users/:id",
								HandlerName: "UpdateUser",
								Parameters: []models.Parameter{
									{
										Name:     "id",
										Type:     "string",
										Source:   models.ParameterSourcePath,
										Required: true,
									},
									{
										Name:     "ctx",
										Type:     "echo.Context",
										Source:   models.ParameterSourceContext,
										Required: true,
										Position: 1,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			result, err := p.ParseSource("test.go", tt.source)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify package name
			if result.PackageName != tt.expected.PackageName {
				t.Errorf("expected package name %s, got %s", tt.expected.PackageName, result.PackageName)
			}

			// Verify controllers
			if len(result.Controllers) != len(tt.expected.Controllers) {
				t.Errorf("expected %d controllers, got %d", len(tt.expected.Controllers), len(result.Controllers))
				return
			}

			for i, expectedController := range tt.expected.Controllers {
				actualController := result.Controllers[i]

				if actualController.Name != expectedController.Name {
					t.Errorf("controller %d: expected name %s, got %s", i, expectedController.Name, actualController.Name)
				}

				// Verify routes
				if len(actualController.Routes) != len(expectedController.Routes) {
					t.Errorf("controller %d: expected %d routes, got %d", i, len(expectedController.Routes), len(actualController.Routes))
					continue
				}

				for j, expectedRoute := range expectedController.Routes {
					actualRoute := actualController.Routes[j]

					if actualRoute.Method != expectedRoute.Method {
						t.Errorf("route %d.%d: expected method %s, got %s", i, j, expectedRoute.Method, actualRoute.Method)
					}

					if actualRoute.Path != expectedRoute.Path {
						t.Errorf("route %d.%d: expected path %s, got %s", i, j, expectedRoute.Path, actualRoute.Path)
					}

					if actualRoute.HandlerName != expectedRoute.HandlerName {
						t.Errorf("route %d.%d: expected handler %s, got %s", i, j, expectedRoute.HandlerName, actualRoute.HandlerName)
					}

					// Verify flags
					if len(actualRoute.Flags) != len(expectedRoute.Flags) {
						t.Errorf("route %d.%d: expected %d flags, got %d", i, j, len(expectedRoute.Flags), len(actualRoute.Flags))
					} else {
						for k, expectedFlag := range expectedRoute.Flags {
							if actualRoute.Flags[k] != expectedFlag {
								t.Errorf("route %d.%d flag %d: expected %s, got %s", i, j, k, expectedFlag, actualRoute.Flags[k])
							}
						}
					}

					// Verify parameters
					if len(actualRoute.Parameters) != len(expectedRoute.Parameters) {
						t.Errorf("route %d.%d: expected %d parameters, got %d", i, j, len(expectedRoute.Parameters), len(actualRoute.Parameters))
						continue
					}

					for k, expectedParam := range expectedRoute.Parameters {
						actualParam := actualRoute.Parameters[k]

						if actualParam.Name != expectedParam.Name {
							t.Errorf("route %d.%d param %d: expected name %s, got %s", i, j, k, expectedParam.Name, actualParam.Name)
						}

						if actualParam.Type != expectedParam.Type {
							t.Errorf("route %d.%d param %d: expected type %s, got %s", i, j, k, expectedParam.Type, actualParam.Type)
						}

						if actualParam.Source != expectedParam.Source {
							t.Errorf("route %d.%d param %d: expected source %v, got %v", i, j, k, expectedParam.Source, actualParam.Source)
						}

						if actualParam.Position != expectedParam.Position {
							t.Errorf("route %d.%d param %d: expected position %d, got %d", i, j, k, expectedParam.Position, actualParam.Position)
						}
					}
				}
			}
		})
	}
}

func TestAdvancedRouteFeatures_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "empty echo parameter name",
			source: `package test

//axon::controller
type UserController struct{}

//axon::route GET /users/:
func (uc *UserController) GetUsers() {}`,
			expectError: true,
			errorMsg:    "parameter name cannot be empty",
		},
		{
			name: "mixed valid and invalid parameters",
			source: `package test

//axon::controller
type UserController struct{}

//axon::route GET /users/{id:int}/posts/:
func (uc *UserController) GetUserPosts() {}`,
			expectError: true,
			errorMsg:    "parameter name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.ParseSource("test.go", tt.source)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}