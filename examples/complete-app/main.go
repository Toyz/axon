package main

import (
	"context"
	"fmt"
	"log"
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
		
		// invoke echo
		fx.Invoke(func(lc fx.Lifecycle, e *echo.Echo, cfg *config.Config) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go e.Start(fmt.Sprintf(":%d", cfg.Port))
					return nil
				},
				OnStop: func(ctx context.Context) error {
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