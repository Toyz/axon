package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/toyz/axon/examples/complete-app/internal/config"
	"github.com/toyz/axon/examples/complete-app/internal/models"
)

//axon::core -Init
type UserService struct {
	//axon::inject
	Config *config.Config
	users  map[int]*models.User
	nextID int
	mu     sync.RWMutex
}

// Start initializes the user service
func (s *UserService) Start(ctx context.Context) error {
	s.users = make(map[int]*models.User)
	s.nextID = 1
	
	// Add some sample data
	s.users[1] = &models.User{
		ID:        1,
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}
	s.users[2] = &models.User{
		ID:        2,
		Name:      "Jane Smith",
		Email:     "jane@example.com",
		CreatedAt: time.Now(),
	}
	s.nextID = 3
	
	fmt.Printf("UserService started with %d users\n", len(s.users))
	return nil
}

// Stop cleans up the user service
func (s *UserService) Stop(ctx context.Context) error {
	fmt.Println("UserService stopped")
	return nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(id int) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	user, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user with id %d not found", id)
	}
	return user, nil
}

// GetAllUsers retrieves all users
func (s *UserService) GetAllUsers() ([]*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	users := make([]*models.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users, nil
}

// CreateUser creates a new user
func (s *UserService) CreateUser(req models.CreateUserRequest) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	user := &models.User{
		ID:        s.nextID,
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: time.Now(),
	}
	
	if err := user.Validate(); err != nil {
		return nil, err
	}
	
	s.users[s.nextID] = user
	s.nextID++
	
	return user, nil
}

// UpdateUser updates an existing user
func (s *UserService) UpdateUser(id int, req models.UpdateUserRequest) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	user, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user with id %d not found", id)
	}
	
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	
	if err := user.Validate(); err != nil {
		return nil, err
	}
	
	return user, nil
}

// DeleteUser deletes a user by ID
func (s *UserService) DeleteUser(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.users[id]; !exists {
		return fmt.Errorf("user with id %d not found", id)
	}
	
	delete(s.users, id)
	return nil
}

// SearchUsers searches for users based on criteria
func (s *UserService) SearchUsers(name string, age int, active bool) ([]*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var results []*models.User
	
	for _, user := range s.users {
		// Simple search logic - in real app this would be more sophisticated
		if name != "" && !strings.Contains(strings.ToLower(user.Name), strings.ToLower(name)) {
			continue
		}
		
		// For this example, we'll just return matching users
		// In a real app, you'd have age and active fields in the User model
		results = append(results, user)
	}
	
	return results, nil
}