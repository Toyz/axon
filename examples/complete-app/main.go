package main

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
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/toyz/axon/examples/complete-app/internal/config"
	"github.com/toyz/axon/examples/complete-app/internal/controllers"
	"github.com/toyz/axon/examples/complete-app/internal/interfaces"
	"github.com/toyz/axon/examples/complete-app/internal/middleware"
	"github.com/toyz/axon/examples/complete-app/internal/services"
	"github.com/toyz/axon/examples/complete-app/internal/logging"
	"github.com/toyz/axon/pkg/axon"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		// Provide configuration
		fx.Provide(config.LoadConfig),
		
		// Provide Echo instance
		fx.Provide(func() *echo.Echo {
			e := echo.New()
			e.Use(echomiddleware.Recover())
			e.Use(echomiddleware.CORS())
			return e
		}),
		
		// Include generated modules
		controllers.AutogenModule,
		services.AutogenModule,
		middleware.AutogenModule,
		interfaces.AutogenModule,
		logging.AutogenModule,
		
		// Start the HTTP server
		fx.Invoke(func(lc fx.Lifecycle, e *echo.Echo, cfg *config.Config) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					// Register routes from axon registry
					routes := axon.GetRoutes()
					for _, route := range routes {
						if route.Handler != nil {
							e.Add(route.Method, route.EchoPath, route.Handler)
						}
					}
					
					// Start server in a goroutine
					go func() {
						addr := fmt.Sprintf(":%d", cfg.Port)
						fmt.Printf("Starting server on %s\n", addr)
						if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
							log.Fatalf("Server failed to start: %v", err)
						}
					}()
					
					return nil
				},
				OnStop: func(ctx context.Context) error {
					fmt.Println("Shutting down server...")
					return e.Shutdown(ctx)
				},
			})
		}),
	)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the application
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("Received shutdown signal")

	// Stop the application with timeout
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()

	if err := app.Stop(stopCtx); err != nil {
		log.Fatalf("Failed to stop application gracefully: %v", err)
	}

	fmt.Println("Application stopped")
}