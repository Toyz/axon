package axon

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// ServerConfig holds configuration for the Axon web server
type ServerConfig struct {
	// Port is the port to listen on (default: 8080)
	Port string

	// Host is the host to bind to (default: "")
	Host string

	// EnableCORS enables CORS middleware (default: true)
	EnableCORS bool

	// EnableLogger enables request logging middleware (default: true)
	EnableLogger bool

	// EnableRecover enables panic recovery middleware (default: true)
	EnableRecover bool

	// ShutdownTimeout is the timeout for graceful shutdown (default: 30s)
	ShutdownTimeout time.Duration
}

// DefaultServerConfig returns a server configuration with sensible defaults
func DefaultServerConfig() *ServerConfig {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &ServerConfig{
		Port:            port,
		Host:            "",
		EnableCORS:      true,
		EnableLogger:    true,
		EnableRecover:   true,
		ShutdownTimeout: 30 * time.Second,
	}
}

// Server wraps an Echo instance with additional Axon functionality
type Server struct {
	echo   *echo.Echo
	config *ServerConfig
}

// NewServer creates a new Axon server with the given configuration
func NewServer(config *ServerConfig) *Server {
	if config == nil {
		config = DefaultServerConfig()
	}

	e := echo.New()

	// Configure Echo
	e.HideBanner = true
	e.HidePort = false

	// Add default middleware
	if config.EnableRecover {
		e.Use(middleware.Recover())
	}

	if config.EnableLogger {
		e.Use(middleware.Logger())
	}

	if config.EnableCORS {
		e.Use(middleware.CORS())
	}

	return &Server{
		echo:   e,
		config: config,
	}
}

// Echo returns the underlying Echo instance for advanced configuration
func (s *Server) Echo() *echo.Echo {
	return s.echo
}

// RegisterRoutes registers all routes from the global route registry
func (s *Server) RegisterRoutes() {
	// For now, register routes directly with Echo
	// TODO: Refactor this to use WebServerInterface abstraction
	routes := DefaultRouteRegistry.GetAllRoutes()
	for _, route := range routes {
		// Convert back to Echo handler for now
		echoHandler := func(c echo.Context) error {
			// This is a temporary bridge - we need to create proper adapters
			return nil
		}

		// Use EchoPath for registration with Echo (converts {id:int} to :id)
		s.echo.Add(route.Method, route.EchoPath, echoHandler)
	}
}

// Start starts the server and blocks until shutdown
func (s *Server) Start() error {
	// Register all routes
	s.RegisterRoutes()

	// Start server in a goroutine
	go func() {
		addr := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)
		log.Printf("Starting server on %s", addr)
		if err := s.echo.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	// Shutdown server
	if err := s.echo.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server shutdown complete")
	return nil
}

// StartWithFX starts the server using the FX lifecycle (for use with generated main.go)
func (s *Server) StartWithFX(lc interface{}) {
	// This would be implemented to work with fx.Lifecycle
	// For now, just start normally
	if err := s.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
