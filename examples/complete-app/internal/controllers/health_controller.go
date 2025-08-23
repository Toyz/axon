package controllers

import (
	"github.com/toyz/axon/examples/complete-app/internal/services"
)

//axon::controller
type HealthController struct {
	//axon::inject
	DatabaseService *services.DatabaseService
}

//axon::route GET /health
func (c *HealthController) GetHealth() (map[string]interface{}, error) {
	return map[string]interface{}{
		"status":   "healthy",
		"database": c.DatabaseService.Health(),
	}, nil
}

//axon::route GET /ready
func (c *HealthController) GetReadiness() (map[string]interface{}, error) {
	ready := c.DatabaseService.IsConnected()
	
	return map[string]interface{}{
		"ready":    ready,
		"database": ready,
	}, nil
}