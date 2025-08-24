package parser

import (
	"strings"
	"testing"

	"github.com/toyz/axon/internal/models"
)

func TestParser_CustomTypeParameterParsing(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		expectedRoutes int
		expectedParams []struct {
			name         string
			paramType    string
			isCustomType bool
		}
		expectError bool
		errorMsg    string
	}{
		{
			name: "route with built-in UUID type",
			source: `
package test

import "github.com/labstack/echo/v4"

//axon::controller
type UserController struct {}

//axon::route GET /users/{id:uuid.UUID}
func (c *UserController) GetUser(id string) error {
	return nil
}
`,
			expectedRoutes: 1,
			expectedParams: []struct {
				name         string
				paramType    string
				isCustomType bool
			}{
				{name: "id", paramType: "uuid.UUID", isCustomType: false},
			},
		},
		{
			name: "route with custom type and built-in type",
			source: `
package test

import "github.com/labstack/echo/v4"

//axon::controller
type UserController struct {}

//axon::route GET /users/{id:CustomID}/posts/{count:int}
func (c *UserController) GetUserPosts(id CustomID, count int) error {
	return nil
}

type CustomID string

//axon::route_parser CustomID
func ParseCustomID(c echo.Context, paramValue string) (CustomID, error) {
	return CustomID(paramValue), nil
}
`,
			expectedRoutes: 1,
			expectedParams: []struct {
				name         string
				paramType    string
				isCustomType bool
			}{
				{name: "id", paramType: "CustomID", isCustomType: true},
				{name: "count", paramType: "int", isCustomType: false},
			},
		},
		{
			name: "route with multiple custom types",
			source: `
package test

import "github.com/labstack/echo/v4"

//axon::controller
type UserController struct {}

//axon::route GET /orgs/{orgId:OrgID}/users/{userId:UserID}
func (c *UserController) GetOrgUser(orgId OrgID, userId UserID) error {
	return nil
}

type OrgID string
type UserID string

//axon::route_parser OrgID
func ParseOrgID(c echo.Context, paramValue string) (OrgID, error) {
	return OrgID(paramValue), nil
}

//axon::route_parser UserID
func ParseUserID(c echo.Context, paramValue string) (UserID, error) {
	return UserID(paramValue), nil
}
`,
			expectedRoutes: 1,
			expectedParams: []struct {
				name         string
				paramType    string
				isCustomType bool
			}{
				{name: "orgId", paramType: "OrgID", isCustomType: true},
				{name: "userId", paramType: "UserID", isCustomType: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			metadata, err := p.ParseSource("test.go", tt.source)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(metadata.Controllers) != 1 {
				t.Errorf("expected 1 controller, got %d", len(metadata.Controllers))
				return
			}

			controller := metadata.Controllers[0]
			if len(controller.Routes) != tt.expectedRoutes {
				t.Errorf("expected %d routes, got %d", tt.expectedRoutes, len(controller.Routes))
				return
			}

			route := controller.Routes[0]
			if len(route.Parameters) != len(tt.expectedParams) {
				t.Errorf("expected %d parameters, got %d", len(tt.expectedParams), len(route.Parameters))
				return
			}

			for i, expectedParam := range tt.expectedParams {
				param := route.Parameters[i]
				if param.Name != expectedParam.name {
					t.Errorf("param %d: expected name '%s', got '%s'", i, expectedParam.name, param.Name)
				}
				if param.Type != expectedParam.paramType {
					t.Errorf("param %d: expected type '%s', got '%s'", i, expectedParam.paramType, param.Type)
				}
				if param.IsCustomType != expectedParam.isCustomType {
					t.Errorf("param %d: expected IsCustomType %v, got %v", i, expectedParam.isCustomType, param.IsCustomType)
				}
				if param.Source != models.ParameterSourcePath {
					t.Errorf("param %d: expected source %v, got %v", i, models.ParameterSourcePath, param.Source)
				}
			}
		})
	}
}

func TestParser_CustomTypeValidation(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "route with custom type but no parser registered",
			source: `
package test

import "github.com/labstack/echo/v4"

//axon::controller
type UserController struct {}

//axon::route GET /users/{id:CustomID}
func (c *UserController) GetUser(id string) error {
	return nil
}
`,
			expectError: true,
			errorMsg:    "no parser registered for custom type 'CustomID'",
		},
		{
			name: "route with custom type and matching parser",
			source: `
package test

import "github.com/labstack/echo/v4"

//axon::controller
type UserController struct {}

//axon::route GET /users/{id:CustomID}
func (c *UserController) GetUser(id CustomID) error {
	return nil
}

type CustomID string

//axon::route_parser CustomID
func ParseCustomID(c echo.Context, paramValue string) (CustomID, error) {
	return CustomID(paramValue), nil
}
`,
			expectError: false,
		},
		{
			name: "route with built-in UUID type (should work without custom parser)",
			source: `
package test

import (
	"github.com/labstack/echo/v4"
	"github.com/google/uuid"
)

//axon::controller
type UserController struct {}

//axon::route GET /users/{id:uuid.UUID}
func (c *UserController) GetUser(id uuid.UUID) error {
	return nil
}
`,
			expectError: false,
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

func TestParser_CustomTypeParserLinking(t *testing.T) {
	source := `
package test

import "github.com/labstack/echo/v4"

//axon::controller
type UserController struct {}

//axon::route GET /users/{id:CustomID}
func (c *UserController) GetUser(id CustomID) error {
	return nil
}

type CustomID string

//axon::route_parser CustomID
func ParseCustomID(c echo.Context, paramValue string) (CustomID, error) {
	return CustomID(paramValue), nil
}
`

	p := NewParser()
	metadata, err := p.ParseSource("test.go", source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that we have the parser registered
	if len(metadata.RouteParsers) != 1 {
		t.Errorf("expected 1 route parser, got %d", len(metadata.RouteParsers))
		return
	}

	parser := metadata.RouteParsers[0]
	if parser.TypeName != "CustomID" {
		t.Errorf("expected parser type name 'CustomID', got '%s'", parser.TypeName)
	}
	if parser.FunctionName != "ParseCustomID" {
		t.Errorf("expected parser function name 'ParseCustomID', got '%s'", parser.FunctionName)
	}

	// Check that the route parameter is linked to the parser
	if len(metadata.Controllers) != 1 {
		t.Errorf("expected 1 controller, got %d", len(metadata.Controllers))
		return
	}

	controller := metadata.Controllers[0]
	if len(controller.Routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(controller.Routes))
		return
	}

	route := controller.Routes[0]
	if len(route.Parameters) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(route.Parameters))
		return
	}

	param := route.Parameters[0]
	if !param.IsCustomType {
		t.Errorf("expected parameter to be marked as custom type")
	}
	if param.ParserFunc != "ParseCustomID" {
		t.Errorf("expected parameter parser function 'ParseCustomID', got '%s'", param.ParserFunc)
	}
	if param.Type != "CustomID" {
		t.Errorf("expected parameter type 'CustomID', got '%s'", param.Type)
	}
}