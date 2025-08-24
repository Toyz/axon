package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toyz/axon/internal/models"
)

// TestLifecycleModeGeneration tests the generation of different lifecycle modes
func TestLifecycleModeGeneration(t *testing.T) {
	t.Run("SingletonMode", func(t *testing.T) {
		service := models.CoreServiceMetadata{
			Name:       "UserService",
			StructName: "UserService",
			Mode:       "Singleton", // Explicit singleton
			Dependencies: []models.Dependency{
				{Name: "Config", Type: "*config.Config", IsInit: false},
			},
		}

		result, err := GenerateCoreServiceProvider(service)
		require.NoError(t, err)

		// Should generate regular provider, not factory
		assert.Contains(t, result, "func NewUserService(")
		assert.Contains(t, result, "return &UserService{")
		assert.NotContains(t, result, "Factory")
		assert.NotContains(t, result, "func() *UserService")
	})

	t.Run("TransientMode", func(t *testing.T) {
		service := models.CoreServiceMetadata{
			Name:       "SessionService",
			StructName: "SessionService",
			Mode:       "Transient", // Transient mode
			Dependencies: []models.Dependency{
				{Name: "DatabaseService", Type: "*DatabaseService", IsInit: false},
			},
		}

		result, err := GenerateCoreServiceProvider(service)
		require.NoError(t, err)

		// Should generate factory function
		assert.Contains(t, result, "func NewSessionServiceFactory(")
		assert.Contains(t, result, "func() *SessionService")
		assert.Contains(t, result, "return func() *SessionService {")
		assert.Contains(t, result, "return &SessionService{")
		assert.Contains(t, result, "DatabaseService: DatabaseService,")
	})

	t.Run("DefaultModeIsSingleton", func(t *testing.T) {
		service := models.CoreServiceMetadata{
			Name:       "DefaultService",
			StructName: "DefaultService",
			Mode:       "", // Empty mode should default to singleton
			Dependencies: []models.Dependency{
				{Name: "Config", Type: "*config.Config", IsInit: false},
			},
		}

		result, err := GenerateCoreServiceProvider(service)
		require.NoError(t, err)

		// Should generate regular provider (singleton behavior)
		assert.Contains(t, result, "func NewDefaultService(")
		assert.Contains(t, result, "return &DefaultService{")
		assert.NotContains(t, result, "Factory")
	})

	t.Run("TransientWithLifecycle", func(t *testing.T) {
		service := models.CoreServiceMetadata{
			Name:         "TransientLifecycleService",
			StructName:   "TransientLifecycleService",
			Mode:         "Transient",
			HasLifecycle: true,
			HasStart:     true,
			HasStop:      true,
			Dependencies: []models.Dependency{
				{Name: "Config", Type: "*config.Config", IsInit: false},
			},
		}

		result, err := GenerateCoreServiceProvider(service)
		require.NoError(t, err)

		// Transient mode should override lifecycle - generate factory
		assert.Contains(t, result, "func NewTransientLifecycleServiceFactory(")
		assert.Contains(t, result, "func() *TransientLifecycleService")
		assert.NotContains(t, result, "lc fx.Lifecycle") // No lifecycle in transient
	})

	t.Run("TransientWithNoDependencies", func(t *testing.T) {
		service := models.CoreServiceMetadata{
			Name:         "SimpleTransientService",
			StructName:   "SimpleTransientService",
			Mode:         "Transient",
			Dependencies: []models.Dependency{}, // No dependencies
		}

		result, err := GenerateCoreServiceProvider(service)
		require.NoError(t, err)

		// Should generate factory with no parameters
		assert.Contains(t, result, "func NewSimpleTransientServiceFactory() func() *SimpleTransientService")
		assert.Contains(t, result, "return func() *SimpleTransientService {")
		assert.Contains(t, result, "return &SimpleTransientService{")
	})
}

// TestLifecycleModeModuleGeneration tests module generation with different lifecycle modes
func TestLifecycleModeModuleGeneration(t *testing.T) {
	metadata := &models.PackageMetadata{
		PackageName: "services",
		PackagePath: "./services",
		CoreServices: []models.CoreServiceMetadata{
			{
				Name:       "UserService",
				StructName: "UserService",
				Mode:       "Singleton",
				Dependencies: []models.Dependency{
					{Name: "Config", Type: "*config.Config", IsInit: false},
				},
			},
			{
				Name:       "SessionService",
				StructName: "SessionService",
				Mode:       "Transient",
				Dependencies: []models.Dependency{
					{Name: "DatabaseService", Type: "*DatabaseService", IsInit: false},
				},
			},
		},
	}

	result, err := GenerateCoreServiceModuleWithModule(metadata, "github.com/example/app")
	require.NoError(t, err)

	// Should contain both singleton and transient providers
	assert.Contains(t, result, "func NewUserService(")
	assert.Contains(t, result, "func NewSessionServiceFactory(")

	// Module should register them differently
	assert.Contains(t, result, "fx.Provide(NewUserService),")        // Singleton
	assert.Contains(t, result, "fx.Provide(NewSessionServiceFactory),") // Transient factory
}

// TestInvalidLifecycleMode tests error handling for invalid modes
func TestInvalidLifecycleMode(t *testing.T) {
	service := models.CoreServiceMetadata{
		Name:       "InvalidService",
		StructName: "InvalidService",
		Mode:       "InvalidMode", // Invalid mode
		Dependencies: []models.Dependency{
			{Name: "Config", Type: "*config.Config", IsInit: false},
		},
	}

	// This should still work since validation happens in the parser, not template
	result, err := GenerateCoreServiceProvider(service)
	require.NoError(t, err)

	// Should fall back to singleton behavior for unknown modes
	assert.Contains(t, result, "func NewInvalidService(")
	assert.NotContains(t, result, "Factory")
}