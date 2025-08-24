package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/toyz/axon/examples/complete-app/internal/services"
	"github.com/toyz/axon/pkg/axon"
)

// SessionController demonstrates using a Transient service
// Each request gets its own SessionService instance via the factory
//axon::controller
type SessionController struct {
	//axon::inject
	SessionFactory func() *services.SessionService // Inject the factory function for transient service
	//axon::inject
	UserService    *services.UserService          // Regular singleton service
}

// StartSession creates a new session for a user
//axon::route POST /sessions/{userID:int} -Middleware=LoggingMiddleware
func (c *SessionController) StartSession(userID int) (*axon.Response, error) {
	// Get a fresh SessionService instance for this request
	sessionService := c.SessionFactory()
	
	// Verify user exists using the singleton UserService
	user, err := c.UserService.GetUser(userID)
	if err != nil {
		return &axon.Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"error": "User not found"},
		}, nil
	}
	
	// Start a new session with the fresh service instance
	sessionID := sessionService.StartSession(userID)
	
	return &axon.Response{
		StatusCode: http.StatusCreated,
		Body: map[string]interface{}{
			"session_id": sessionID,
			"user":       user,
			"message":    "Session started successfully",
		},
	}, nil
}

// GetSessionInfo returns information about the current session
// This demonstrates that each request gets its own session instance
//axon::route GET /sessions/info/{userID:int} -Middleware=LoggingMiddleware
func (c *SessionController) GetSessionInfo(userID int) (map[string]interface{}, error) {
	// Get a fresh SessionService instance for this request
	sessionService := c.SessionFactory()
	
	// Start a session to demonstrate the transient behavior
	sessionID := sessionService.StartSession(userID)
	
	// Get session info from this specific instance
	info := sessionService.GetSessionInfo()
	
	// Add some metadata to show this is a fresh instance
	info["is_fresh_instance"] = true
	info["session_id"] = sessionID
	info["note"] = "This is a new SessionService instance created just for this request"
	
	return info, nil
}

// CompareSessionInstances demonstrates that different requests get different instances
//axon::route GET /sessions/compare -PassContext
func (c *SessionController) CompareSessionInstances(ctx echo.Context) error {
	// Create multiple session instances to show they're different
	session1 := c.SessionFactory()
	session2 := c.SessionFactory()
	session3 := c.SessionFactory()
	
	// Start sessions with different user IDs
	id1 := session1.StartSession(1)
	id2 := session2.StartSession(2)
	id3 := session3.StartSession(3)
	
	response := map[string]interface{}{
		"message": "Each call to SessionFactory() creates a new instance",
		"instances": []map[string]interface{}{
			{
				"instance": "session1",
				"session_id": id1,
				"info": session1.GetSessionInfo(),
			},
			{
				"instance": "session2", 
				"session_id": id2,
				"info": session2.GetSessionInfo(),
			},
			{
				"instance": "session3",
				"session_id": id3,
				"info": session3.GetSessionInfo(),
			},
		},
		"note": "Notice how each instance has different session IDs and timestamps",
	}
	
	return ctx.JSON(http.StatusOK, response)
}