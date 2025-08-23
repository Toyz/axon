package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/toyz/axon/examples/complete-app/internal/models"
	"github.com/toyz/axon/examples/complete-app/internal/services"
	"github.com/toyz/axon/pkg/axon"
)

//axon::controller
type UserController struct {
	//axon::inject
	UserService *services.UserService
}

//axon::route GET /users
func (c *UserController) GetAllUsers() ([]*models.User, error) {
	return c.UserService.GetAllUsers()
}

//axon::route GET /users/{id:int}
func (c *UserController) GetUser(id int) (*models.User, error) {
	return c.UserService.GetUser(id)
}

//axon::route POST /users
func (c *UserController) CreateUser(req models.CreateUserRequest) (*axon.Response, error) {
	user, err := c.UserService.CreateUser(req)
	if err != nil {
		return &axon.Response{
			StatusCode: http.StatusBadRequest,
			Body:       map[string]string{"error": err.Error()},
		}, nil
	}
	
	return &axon.Response{
		StatusCode: http.StatusCreated,
		Body:       user,
	}, nil
}

//axon::route PUT /users/{id:int}
func (c *UserController) UpdateUser(id int, req models.UpdateUserRequest) (*axon.Response, error) {
	user, err := c.UserService.UpdateUser(id, req)
	if err != nil {
		return &axon.Response{
			StatusCode: http.StatusBadRequest,
			Body:       map[string]string{"error": err.Error()},
		}, nil
	}
	
	return &axon.Response{
		StatusCode: http.StatusOK,
		Body:       user,
	}, nil
}

//axon::route DELETE /users/{id:int} -PassContext
func (c *UserController) DeleteUser(ctx echo.Context, id int) error {
	err := c.UserService.DeleteUser(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	
	return ctx.NoContent(http.StatusNoContent)
}