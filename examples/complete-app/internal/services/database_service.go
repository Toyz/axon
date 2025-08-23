package services

import (
	"context"
	"fmt"
	"time"

	"github.com/toyz/axon/examples/complete-app/internal/config"
)

//axon::core -Init
type DatabaseService struct {
	//axon::inject
	Config *config.Config
	connected bool
}

// Start initializes the database connection
func (s *DatabaseService) Start(ctx context.Context) error {
	fmt.Printf("Connecting to database: %s\n", s.Config.DatabaseURL)
	
	// Simulate connection time
	time.Sleep(100 * time.Millisecond)
	
	s.connected = true
	fmt.Println("Database connection established")
	return nil
}

// Stop closes the database connection
func (s *DatabaseService) Stop(ctx context.Context) error {
	if s.connected {
		fmt.Println("Closing database connection")
		s.connected = false
	}
	return nil
}

// IsConnected returns whether the database is connected
func (s *DatabaseService) IsConnected() bool {
	return s.connected
}

// Health returns the health status of the database
func (s *DatabaseService) Health() map[string]interface{} {
	return map[string]interface{}{
		"connected":    s.connected,
		"database_url": s.Config.DatabaseURL,
		"status":       "healthy",
	}
}

