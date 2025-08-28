package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/toyz/axon/examples/complete-app/internal/models"
	"github.com/toyz/axon/examples/complete-app/internal/services"
	"github.com/toyz/axon/pkg/axon"
)

//axon::controller -Prefix=/api/v1/users/{userId:int} -Middleware=AuthMiddleware
type UserController struct {
	//axon::inject
	UserService *services.UserService
}

//axon::route GET /
func (c *UserController) GetAllUsers() ([]*models.User, error) {
	return c.UserService.GetAllUsers()
}

//axon::route GET /search
func (c *UserController) SearchUsers(ctx echo.Context, query axon.QueryMap) ([]*models.User, error) {
	// Access query parameters easily
	name := query.Get("name")
	age := query.GetInt("age")
	active := query.GetBool("active")
	
	return c.UserService.SearchUsers(name, age, active)
}

//axon::route GET /profile
func (c *UserController) GetUser(userId int) (*models.User, error) {
	user, err := c.UserService.GetUser(userId)
	if err != nil {
		// Example of using axon.HttpError for better error responses
		return nil, axon.ErrNotFound("User not found")
	}
	return user, nil
}

//axon::route POST /users -Middleware=AuthMiddleware
func (c *UserController) CreateUser(req models.CreateUserRequest) (*axon.Response, error) {
	user, err := c.UserService.CreateUser(req)
	if err != nil {
		return axon.BadRequest(err.Error()), nil
	}
	
	// Example of using enhanced Response with headers and cookies
	return axon.Created(user).
		WithHeader("Location", "/users/"+string(rune(user.ID))).
		WithHeader("X-Created-At", user.CreatedAt.Format("2006-01-02T15:04:05Z")).
		WithSimpleCookie("last-created-user", string(rune(user.ID))), nil
}

//axon::route PUT /users/{id:int} -Middleware=AuthMiddleware
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

//axon::route DELETE /users/{id:int} -Middleware=AuthMiddleware -PassContext
func (c *UserController) DeleteUser(ctx echo.Context, id int) error {
	err := c.UserService.DeleteUser(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	
	return ctx.NoContent(http.StatusNoContent)
}