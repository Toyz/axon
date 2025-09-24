package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/toyz/axon/examples/complete-app/internal/config"
	"github.com/toyz/axon/examples/complete-app/internal/controllers"
	"github.com/toyz/axon/examples/complete-app/internal/interfaces"
	"github.com/toyz/axon/examples/complete-app/internal/logging"
	"github.com/toyz/axon/examples/complete-app/internal/middleware"
	"github.com/toyz/axon/examples/complete-app/internal/services"
	"github.com/toyz/axon/pkg/axon"
	"github.com/toyz/axon/pkg/axon/adapters"
	"go.uber.org/fx"
)

func main() {
	// Define CLI flags
	var adapter = flag.String("adapter", "echo", "Web server adapter to use (echo, gin, or fiber)")
	var port = flag.Int("port", 8080, "Port to run the server on")
	var help = flag.Bool("help", false, "Show help information")
	flag.Parse()

	if *help {
		fmt.Println("Complete App - Axon Framework Demo")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Printf("  %s [options]\n", os.Args[0])
		fmt.Println("")
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Printf("  %s -adapter=echo -port=8080\n", os.Args[0])
		fmt.Printf("  %s -adapter=gin -port=3000\n", os.Args[0])
		fmt.Printf("  %s -adapter=fiber -port=3000\n", os.Args[0])
		os.Exit(0)
	}

	// Validate adapter choice
	if *adapter != "echo" && *adapter != "gin" && *adapter != "fiber" {
		log.Fatalf("Invalid adapter '%s'. Must be 'echo', 'gin', or 'fiber'", *adapter)
	}

	fmt.Printf("🚀 Starting Complete App with %s adapter on port %d\n", *adapter, *port)

	app := fx.New(
		// Provide configuration with command line overrides
		fx.Provide(func() *config.Config {
			cfg := config.LoadConfig()
			if *port != 8080 { // Only override if user specified a different port
				cfg.Port = *port
			}
			return cfg
		}),

		// Provide selected adapter as WebServerInterface
		fx.Provide(func() axon.WebServerInterface {
			switch *adapter {
			case "gin":
				fmt.Println("📦 Using Gin web framework")
				return adapters.NewDefaultGinAdapter()
			case "echo":
				fmt.Println("📦 Using Echo web framework")
				return adapters.NewDefaultEchoAdapter()
			case "fiber":
				fmt.Println("📦 Using Fiber web framework")
				return adapters.NewDefaultFiberAdapter()
			default:
				// This should never happen due to validation above
				panic(fmt.Sprintf("Unknown adapter: %s", *adapter))
			}
		}),

		// Include generated modules
		controllers.AutogenModule,
		services.AutogenModule,
		middleware.AutogenModule,
		interfaces.AutogenModule,
		logging.AutogenModule,

		// invoke server lifecycle
		fx.Invoke(func(lc fx.Lifecycle, server axon.WebServerInterface, cfg *config.Config) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					fmt.Printf("🌐 Starting %s server on http://localhost:%d\n", server.Name(), cfg.Port)
					go server.Start(fmt.Sprintf(":%d", cfg.Port))
					return nil
				},
				OnStop: func(ctx context.Context) error {
					fmt.Printf("⏹️  Stopping %s server...\n", server.Name())
					return server.Stop(ctx)
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
