package interfaces

import (
	"github.com/toyz/axon/examples/complete-app/internal/models"
)

//axom::interface
//axon::core
type UserRepository struct {
	//axon::init
	storage map[int]*models.User
}

// GetUser retrieves a user by ID
func (r *UserRepository) GetUser(id int) (*models.User, error) {
	user, exists := r.storage[id]
	if !exists {
		return nil, nil
	}
	return user, nil
}

// SaveUser saves a user
func (r *UserRepository) SaveUser(user *models.User) error {
	r.storage[user.ID] = user
	return nil
}

// DeleteUser deletes a user
func (r *UserRepository) DeleteUser(id int) error {
	delete(r.storage, id)
	return nil
}

// ListUsers returns all users
func (r *UserRepository) ListUsers() ([]*models.User, error) {
	users := make([]*models.User, 0, len(r.storage))
	for _, user := range r.storage {
		users = append(users, user)
	}
	return users, nil
}