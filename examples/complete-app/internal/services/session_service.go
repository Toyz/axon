package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SessionService represents a user session that should be created fresh for each request
// This is a perfect example of when to use Transient mode - each HTTP request should get
// its own session instance with its own state, rather than sharing a singleton.
//axon::core -Mode=Transient
type SessionService struct {
	//axon::inject
	DatabaseService *DatabaseService
	
	sessionID   string
	createdAt   time.Time
	userID      *int
	isActive    bool
}

// StartSession creates a new session for a user
func (s *SessionService) StartSession(userID int) string {
	s.sessionID = fmt.Sprintf("sess_%s", uuid.New().String()[:8])
	s.createdAt = time.Now()
	s.userID = &userID
	s.isActive = true
	
	fmt.Printf("🔐 Started new session %s for user %d\n", s.sessionID, userID)
	return s.sessionID
}

// GetSessionInfo returns current session information
func (s *SessionService) GetSessionInfo() map[string]interface{} {
	var userID interface{} = nil
	if s.userID != nil {
		userID = *s.userID
	}
	
	return map[string]interface{}{
		"session_id": s.sessionID,
		"created_at": s.createdAt,
		"user_id":    userID,
		"is_active":  s.isActive,
		"duration":   time.Since(s.createdAt).String(),
	}
}

// IsValid checks if the session is still valid (active and not expired)
func (s *SessionService) IsValid() bool {
	if !s.isActive {
		return false
	}
	
	// Sessions expire after 30 minutes
	return time.Since(s.createdAt) < 30*time.Minute
}

// EndSession terminates the current session
func (s *SessionService) EndSession() {
	if s.sessionID != "" {
		fmt.Printf("🔒 Ended session %s\n", s.sessionID)
	}
	s.isActive = false
}

// GetUserID returns the user ID associated with this session
func (s *SessionService) GetUserID() *int {
	if !s.IsValid() {
		return nil
	}
	return s.userID
}