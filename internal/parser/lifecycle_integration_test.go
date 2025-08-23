package parser

import (
	"strings"
	"testing"

	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/internal/templates"
)

// TestLifecycleIntegration tests the complete lifecycle management flow from parsing to code generation
func TestLifecycleIntegration(t *testing.T) {
	source := `package services

import (
	"context"
	"go.uber.org/fx"
)

//axon::core -Init
type DatabaseService struct {
	fx.In
	Config *Config
}

func (s *DatabaseService) Start(ctx context.Context) error {
	// Initialize database connection
	return nil
}

func (s *DatabaseService) Stop(ctx context.Context) error {
	// Close database connection
	return nil
}

//axon::core -Init
type MessageConsumer struct {
	fx.In
	DB     DatabaseService
	Logger Logger
}

func (s *MessageConsumer) Start(ctx context.Context) error {
	// Start consuming messages
	return nil
}

//axon::core
type ConfigService struct {
	fx.In
}

type Config struct {
	DatabaseURL string
}

type Logger interface {
	Info(msg string)
}
`

	parser := NewParser()
	metadata, err := parser.ParseSource("services.go", source)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	// Verify we have the expected core services
	if len(metadata.CoreServices) != 3 {
		t.Errorf("expected 3 core services, got %d", len(metadata.CoreServices))
	}

	// Find each service and verify their properties
	var dbService, msgService, configService *models.CoreServiceMetadata
	for i := range metadata.CoreServices {
		switch metadata.CoreServices[i].Name {
		case "DatabaseService":
			dbService = &metadata.CoreServices[i]
		case "MessageConsumer":
			msgService = &metadata.CoreServices[i]
		case "ConfigService":
			configService = &metadata.CoreServices[i]
		}
	}

	// Verify DatabaseService
	if dbService == nil {
		t.Fatal("DatabaseService not found")
	}
	if !dbService.HasLifecycle {
		t.Error("DatabaseService should have lifecycle")
	}
	if !dbService.HasStart {
		t.Error("DatabaseService should have Start method")
	}
	if !dbService.HasStop {
		t.Error("DatabaseService should have Stop method")
	}

	// Verify MessageConsumer
	if msgService == nil {
		t.Fatal("MessageConsumer not found")
	}
	if !msgService.HasLifecycle {
		t.Error("MessageConsumer should have lifecycle")
	}
	if !msgService.HasStart {
		t.Error("MessageConsumer should have Start method")
	}
	if msgService.HasStop {
		t.Error("MessageConsumer should not have Stop method")
	}

	// Verify ConfigService
	if configService == nil {
		t.Fatal("ConfigService not found")
	}
	if configService.HasLifecycle {
		t.Error("ConfigService should not have lifecycle")
	}

	// Test code generation for lifecycle services
	dbProvider, err := templates.GenerateCoreServiceProvider(*dbService)
	if err != nil {
		t.Fatalf("failed to generate DatabaseService provider: %v", err)
	}

	// Verify the generated provider includes lifecycle hooks
	expectedDbProvider := []string{
		"func NewDatabaseService(lc fx.Lifecycle, Config *Config) *DatabaseService {",
		"lc.Append(fx.Hook{",
		"OnStart: func(ctx context.Context) error {",
		"return service.Start(ctx)",
		"OnStop: func(ctx context.Context) error {",
		"return service.Stop(ctx)",
	}

	for _, expected := range expectedDbProvider {
		if !strings.Contains(dbProvider, expected) {
			t.Errorf("DatabaseService provider missing expected content: %s\n\nGenerated:\n%s", expected, dbProvider)
		}
	}

	// Test MessageConsumer provider (Start only, no Stop)
	msgProvider, err := templates.GenerateCoreServiceProvider(*msgService)
	if err != nil {
		t.Fatalf("failed to generate MessageConsumer provider: %v", err)
	}

	expectedMsgProvider := []string{
		"func NewMessageConsumer(lc fx.Lifecycle, DatabaseService DatabaseService, Logger Logger) *MessageConsumer {",
		"OnStart: func(ctx context.Context) error {",
		"return service.Start(ctx)",
	}

	for _, expected := range expectedMsgProvider {
		if !strings.Contains(msgProvider, expected) {
			t.Errorf("MessageConsumer provider missing expected content: %s\n\nGenerated:\n%s", expected, msgProvider)
		}
	}

	// Verify MessageConsumer provider does NOT include OnStop
	if strings.Contains(msgProvider, "OnStop:") {
		t.Error("MessageConsumer provider should not include OnStop hook")
	}

	// Test ConfigService provider (no lifecycle)
	configProvider, err := templates.GenerateCoreServiceProvider(*configService)
	if err != nil {
		t.Fatalf("failed to generate ConfigService provider: %v", err)
	}

	// Verify ConfigService uses regular provider template
	if strings.Contains(configProvider, "lc fx.Lifecycle") {
		t.Error("ConfigService provider should not include fx.Lifecycle parameter")
	}
	if strings.Contains(configProvider, "lc.Append") {
		t.Error("ConfigService provider should not include lifecycle hooks")
	}

	// Test complete module generation
	moduleCode, err := templates.GenerateCoreServiceModule(metadata)
	if err != nil {
		t.Fatalf("failed to generate module: %v", err)
	}

	expectedModuleContent := []string{
		"package services",
		"import (",
		"\"context\"",
		"\"go.uber.org/fx\"",
		"func NewDatabaseService(",
		"func NewMessageConsumer(",
		"func NewConfigService(",
		"var AutogenModule = fx.Module(",
		"fx.Invoke(NewDatabaseService),",
		"fx.Invoke(NewMessageConsumer),",
		"fx.Provide(NewConfigService),",
	}

	for _, expected := range expectedModuleContent {
		if !strings.Contains(moduleCode, expected) {
			t.Errorf("Module missing expected content: %s\n\nGenerated:\n%s", expected, moduleCode)
		}
	}
}

// TestLifecycleErrorHandling tests error handling during lifecycle management
func TestLifecycleErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		source        string
		expectedError string
	}{
		{
			name: "service with -Init but no Start method",
			source: `package services

//axon::core -Init
type BadService struct {
	fx.In
}`,
			expectedError: "service BadService has -Init flag but missing Start(context.Context) error method",
		},
		{
			name: "service with -Init but wrong Start signature",
			source: `package services

//axon::core -Init
type BadService struct {
	fx.In
}

func (s *BadService) Start() error {
	return nil
}`,
			expectedError: "service BadService has -Init flag but missing Start(context.Context) error method",
		},
		{
			name: "service with -Init but Start returns wrong type",
			source: `package services

//axon::core -Init
type BadService struct {
	fx.In
}

func (s *BadService) Start(ctx context.Context) {
	// No return value
}`,
			expectedError: "service BadService has -Init flag but missing Start(context.Context) error method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			_, err := parser.ParseSource("test.go", tt.source)

			if err == nil {
				t.Error("expected error but got none")
				return
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}