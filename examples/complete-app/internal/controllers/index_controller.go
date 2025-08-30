package controllers

import (
	"fmt"
	"net/http"
	
	"github.com/labstack/echo/v4"
	"github.com/toyz/axon/pkg/axon"
)

// axon::controller -Priority=999
type IndexController struct{}

// axon::route GET / -PassContext
func (*IndexController) Index(ctx echo.Context) (*axon.Response, error) {
	return axon.OK(map[string]string{"message": "Welcome to the index"}), nil
}

// axon::route POST / -PassContext
func (*IndexController) Create(ctx echo.Context) (*axon.Response, error) {
	return axon.Created(map[string]string{"message": "Resource created"}), nil
}

// axon::route GET /{id:string} -PassContext
func (*IndexController) Show(ctx echo.Context) (*axon.Response, error) {
	id := ctx.Param("id")
	return axon.OK(map[string]string{"message": "Resource found", "id": id}), nil
}

// axon::route GET /{id}/fish -PassContext
func (*IndexController) ShowFish(ctx echo.Context) (*axon.Response, error) {
	id := ctx.Param("id")
	return axon.OK(map[string]string{"message": "Fish found", "id": id}), nil
}

// axon::route GET /{id:string}/test
func (*IndexController) ShowTest(ctx echo.Context, id string) (*axon.Response, error) {
	return axon.OK(map[string]string{"message": "Test resource found", "id": id}), nil
}

// axon::route GET /{*} -Priority=999
func (i *IndexController) CatchAll(ctx echo.Context, path string) (*axon.Response, error) {
	return axon.NewResponse(http.StatusNotFound, map[string]interface{}{
		"error":   "Not Found",
		"message": fmt.Sprintf("Route not found: %s", path),
	}), nil
}