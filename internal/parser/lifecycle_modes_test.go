package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLifecycleModeAnnotationParsing tests parsing of -Mode flag in core service annotations
func TestLifecycleModeAnnotationParsing(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		expectedMode   string
		expectError    bool
		errorContains  string
	}{
		{
			name: "DefaultSingletonMode",
			source: `package services

//axon::core
type UserService struct {
	Config *config.Config
}`,
			expectedMode: "Singleton", // Default mode
			expectError:  false,
		},
		{
			name: "ExplicitSingletonMode",
			source: `package services

//axon::core -Mode=Singleton
type UserService struct {
	Config *config.Config
}`,
			expectedMode: "Singleton",
			expectError:  false,
		},
		{
			name: "TransientMode",
			source: `package services

//axon::core -Mode=Transient
type SessionService struct {
	DatabaseService *DatabaseService
}`,
			expectedMode: "Transient",
			expectError:  false,
		},
		{
			name: "InvalidMode",
			source: `package services

//axon::core -Mode=InvalidMode
type BadService struct {
	Config *config.Config
}`,
			expectedMode: "",
			expectError:  true,
			errorContains: "invalid mode 'InvalidMode': must be 'Singleton' or 'Transient'",
		},
		{
			name: "TransientWithLifecycle",
			source: `package services

//axon::core -Mode=Transient -Init
type TransientLifecycleService struct {
	Config *config.Config
}

func (s *TransientLifecycleService) Start(ctx context.Context) error {
	return nil
}`,
			expectedMode: "Transient",
			expectError:  false,
		},
		{
			name: "SingletonWithLifecycle",
			source: `package services

//axon::core -Mode=Singleton -Init
type SingletonLifecycleService struct {
	Config *config.Config
}

func (s *SingletonLifecycleService) Start(ctx context.Context) error {
	return nil
}`,
			expectedMode: "Singleton",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			metadata, err := parser.ParseSource("test.go", tt.source)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}

			require.NoError(t, err)
			require.Len(t, metadata.CoreServices, 1)

			service := metadata.CoreServices[0]
			assert.Equal(t, tt.expectedMode, service.Mode)
		})
	}
}

// TestLifecycleModeWithOtherFlags tests that Mode flag works with other flags
func TestLifecycleModeWithOtherFlags(t *testing.T) {
	source := `package services

//axon::core -Mode=Transient -Manual=CustomModule
type CustomTransientService struct {
	Config *config.Config
}`

	parser := NewParser()
	metadata, err := parser.ParseSource("test.go", source)
	require.NoError(t, err)
	require.Len(t, metadata.CoreServices, 1)

	service := metadata.CoreServices[0]
	assert.Equal(t, "Transient", service.Mode)
	assert.True(t, service.IsManual)
	assert.Equal(t, "CustomModule", service.ModuleName)
}

// TestLifecycleModeConstants tests that the constants are defined correctly
func TestLifecycleModeConstants(t *testing.T) {
	assert.Equal(t, "Singleton", LifecycleModeSingleton)
	assert.Equal(t, "Transient", LifecycleModeTransient)
	assert.Equal(t, "-Mode", FlagMode)
}