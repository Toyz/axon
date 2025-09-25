package models

import (
	"testing"
)

// TestDirectStructureUsage ensures structures work with composition
func TestDirectStructureUsage(t *testing.T) {
	// Test ControllerMetadata using direct field assignment
	controller := &ControllerMetadata{
		BaseMetadataTrait: BaseMetadataTrait{
			Name:       "UserController",
			StructName: "userController",
		},
		PriorityTrait: PriorityTrait{
			Priority: 100,
		},
		MiddlewareTrait: MiddlewareTrait{
			Middlewares: []string{"auth", "logging"},
		},
		Prefix: "/api/users",
		Routes: []RouteMetadata{
			{
				Method:      "GET",
				Path:        "/",
				HandlerName: "GetUsers",
			},
		},
	}

	if controller.GetName() != "UserController" {
		t.Errorf("Expected Name to be 'UserController', got %s", controller.GetName())
	}

	if controller.GetPriority() != 100 {
		t.Errorf("Expected Priority to be 100, got %d", controller.GetPriority())
	}

	// Test CoreServiceMetadata using direct field assignment
	service := &CoreServiceMetadata{
		BaseMetadataTrait: BaseMetadataTrait{
			Name:       "DatabaseService",
			StructName: "databaseService",
		},
		LifecycleTrait: LifecycleTrait{
			HasStart:     true,
			HasStop:      true,
			HasLifecycle: true,
			StartMode:    "Background",
		},
		ManualModuleTrait: ManualModuleTrait{
			IsManual:   true,
			ModuleName: "DatabaseModule",
		},
		ServiceModeTrait: ServiceModeTrait{
			Mode: "Singleton",
		},
		ConstructorTrait: ConstructorTrait{
			Constructor: "NewDatabaseService",
		},
	}

	if service.GetName() != "DatabaseService" {
		t.Errorf("Expected Name to be 'DatabaseService', got %s", service.GetName())
	}

	if !service.HasStartMethod() {
		t.Error("Expected service to have Start method")
	}
}

// TestBuilderPattern ensures builder pattern works correctly
func TestBuilderPattern(t *testing.T) {
	// Test using the builder pattern
	controller := NewMetadataBuilder("UserController", "userController").
		WithPriority(100).
		WithMiddlewares("auth", "logging").
		BuildController("/api/users", []RouteMetadata{
			{
				Method:      "GET",
				Path:        "/",
				HandlerName: "GetUsers",
			},
		})

	if controller.GetName() != "UserController" {
		t.Errorf("Expected Name to be 'UserController', got %s", controller.GetName())
	}

	if controller.GetPriority() != 100 {
		t.Errorf("Expected Priority to be 100, got %d", controller.GetPriority())
	}

	middlewares := controller.GetMiddlewares()
	if len(middlewares) != 2 || middlewares[0] != "auth" || middlewares[1] != "logging" {
		t.Errorf("Expected middlewares [auth, logging], got %v", middlewares)
	}

	// Test core service with lifecycle
	service := NewMetadataBuilder("DatabaseService", "databaseService").
		WithLifecycle(true, true).
		WithStartMode("Background").
		WithManualModule("DatabaseModule").
		WithServiceMode("Singleton").
		WithConstructor("NewDatabaseService").
		BuildCoreService()

	if service.GetName() != "DatabaseService" {
		t.Errorf("Expected Name to be 'DatabaseService', got %s", service.GetName())
	}

	if !service.HasStartMethod() {
		t.Error("Expected service to have Start method")
	}

	if !service.HasStopMethod() {
		t.Error("Expected service to have Stop method")
	}

	if !service.IsLifecycleEnabled() {
		t.Error("Expected lifecycle to be enabled")
	}

	if service.GetStartMode() != "Background" {
		t.Errorf("Expected StartMode to be 'Background', got %s", service.GetStartMode())
	}

	if !service.IsManualModule() {
		t.Error("Expected service to use manual module")
	}

	if service.GetModuleName() != "DatabaseModule" {
		t.Errorf("Expected ModuleName to be 'DatabaseModule', got %s", service.GetModuleName())
	}

	if service.GetServiceMode() != "Singleton" {
		t.Errorf("Expected ServiceMode to be 'Singleton', got %s", service.GetServiceMode())
	}

	if service.GetConstructor() != "NewDatabaseService" {
		t.Errorf("Expected Constructor to be 'NewDatabaseService', got %s", service.GetConstructor())
	}
}

// TestStructureFieldAccess ensures field access works properly
func TestStructureFieldAccess(t *testing.T) {
	// Create structure using builder
	controller := NewMetadataBuilder("TestController", "testController").
		WithDependencies(Dependency{Name: "service", Type: "TestService"}).
		WithPriority(50).
		WithMiddlewares("test").
		BuildController("/test", []RouteMetadata{
			{Method: "POST", Path: "/create", HandlerName: "Create"},
		})

	// Test interface method access
	if controller.GetName() != "TestController" {
		t.Errorf("Expected Name to be TestController, got %s", controller.GetName())
	}

	if controller.GetStructName() != "testController" {
		t.Errorf("Expected StructName to be testController, got %s", controller.GetStructName())
	}

	if len(controller.GetDependencies()) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(controller.GetDependencies()))
	}

	if controller.GetPriority() != 50 {
		t.Errorf("Expected Priority to be 50, got %d", controller.GetPriority())
	}

	// Test direct field access
	if controller.Prefix != "/test" {
		t.Errorf("Expected Prefix to be /test, got %s", controller.Prefix)
	}

	if len(controller.Routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(controller.Routes))
	}
}

// TestDefaultValues ensures default values work correctly
func TestDefaultValues(t *testing.T) {
	// Test default service mode
	service := NewMetadataBuilder("TestService", "testService").
		BuildCoreService()

	if service.GetServiceMode() != "Singleton" {
		t.Errorf("Expected default ServiceMode to be 'Singleton', got %s", service.GetServiceMode())
	}

	// Test default start mode
	serviceWithLifecycle := NewMetadataBuilder("TestService", "testService").
		WithLifecycle(true, false).
		BuildCoreService()

	if serviceWithLifecycle.GetStartMode() != "Same" {
		t.Errorf("Expected default StartMode to be 'Same', got %s", serviceWithLifecycle.GetStartMode())
	}
}

// TestInterfaceImplementation ensures all structures implement expected interfaces
func TestInterfaceImplementation(t *testing.T) {
	// Test that structures implement Metadata interface
	var _ Metadata = &ControllerMetadata{}
	var _ Metadata = &CoreServiceMetadata{}
	var _ Metadata = &LoggerMetadata{}
	var _ Metadata = &ServiceMetadata{}
	var _ Metadata = &MiddlewareMetadata{}
	var _ Metadata = &InterfaceMetadata{}

	// Test that structures with lifecycle implement LifecycleAware
	var _ LifecycleAware = &CoreServiceMetadata{}
	var _ LifecycleAware = &LoggerMetadata{}
	var _ LifecycleAware = &ServiceMetadata{}

	// Test that structures with priority implement PriorityAware
	var _ PriorityAware = &ControllerMetadata{}
	var _ PriorityAware = &MiddlewareMetadata{}

	// Test that structures with manual module implement ManualModuleAware
	var _ ManualModuleAware = &CoreServiceMetadata{}
	var _ ManualModuleAware = &LoggerMetadata{}

	// Test that structures with middleware implement MiddlewareAware
	var _ MiddlewareAware = &ControllerMetadata{}

	// Test that structures with path implement PathAware
	var _ PathAware = &MiddlewareMetadata{}
	var _ PathAware = &InterfaceMetadata{}
}