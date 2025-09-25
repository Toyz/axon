package controllers

import (
	"github.com/toyz/axon/pkg/axon"
)

// axon::controller -Priority=999
type IndexController struct{}

// axon::route GET / -PassContext
func (*IndexController) Index(ctx axon.RequestContext) (*axon.Response, error) {
	return axon.OK(map[string]string{"message": "Welcome to the index"}), nil
}

// axon::route POST / -PassContext
func (*IndexController) Create(ctx axon.RequestContext) (*axon.Response, error) {
	return axon.Created(map[string]string{"message": "Resource created"}), nil
}

// axon::route GET /{id:string} -PassContext
func (*IndexController) Show(ctx axon.RequestContext) (*axon.Response, error) {
	id := ctx.Param("id")
	return axon.OK(map[string]string{"message": "Resource found", "id": id}), nil
}

// axon::route GET /{id}/fish -PassContext
func (*IndexController) ShowFish(ctx axon.RequestContext) (*axon.Response, error) {
	id := ctx.Param("id")
	return axon.OK(map[string]string{"message": "Fish found", "id": id}), nil
}

// axon::route GET /{id:string}/test
func (*IndexController) ShowTest(ctx axon.RequestContext, id string) (*axon.Response, error) {
	return axon.OK(map[string]string{"message": "Test resource found", "id": id}), nil
}

/*
// axon::route GET /{*} -Priority=999
func (i *IndexController) CatchAll(ctx axon.RequestContext, path string) (*axon.Response, error) {
	return axon.NewResponse(http.StatusNotFound, map[string]interface{}{
		"error":   "Not Found",
		"message": fmt.Sprintf("Route not found: %s", path),
	}), nil
}*/
