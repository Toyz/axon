package controllers

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/toyz/axon/pkg/axon"
	"github.com/toyz/axon/examples/complete-app/internal/models"
	"github.com/toyz/axon/examples/complete-app/internal/parsers"
	"github.com/toyz/axon/examples/complete-app/internal/services"
)

//axon::controller
type ProductController struct {
	//axon::inject
	DatabaseService *services.DatabaseService
}

// Using built-in UUID parser (this will use the built-in uuid.UUID parser)
//axon::route GET /products/{id:UUID}
func (c *ProductController) GetProduct(id uuid.UUID) (*models.Product, error) {
	// Mock implementation - in real app this would query the database
	product := &models.Product{
		ID:          id,
		Name:        "Sample Product",
		Description: "A sample product with UUID",
		Price:       99.99,
	}
	return product, nil
}

// Using custom ProductCode parser with middleware
//axon::route GET /products/by-code/{code:ProductCode} -Middleware=LoggingMiddleware
func (c *ProductController) GetProductByCode(code parsers.ProductCode) (*models.Product, error) {
	// Mock implementation showing custom parser usage
	product := &models.Product{
		ID:          uuid.New(),
		Name:        "Product " + code.String(),
		Description: "Product found by code: " + code.String(),
		Price:       149.99,
	}
	return product, nil
}

// Using custom DateRange parser with multiple middleware
//axon::route GET /products/sales/{dateRange:DateRange} -Middleware=AuthMiddleware,LoggingMiddleware
func (c *ProductController) GetProductSales(dateRange parsers.DateRange) ([]models.Product, error) {
	// Mock implementation showing custom date range parser
	products := []models.Product{
		{
			ID:          uuid.New(),
			Name:        "Sales Report Product",
			Description: fmt.Sprintf("Sales from %s to %s", dateRange.Start.Format("2006-01-02"), dateRange.End.Format("2006-01-02")),
			Price:       199.99,
		},
	}
	return products, nil
}

// Mixed parameter types with auth middleware and PassContext
//axon::route POST /products/{categoryId:UUID}/items -Middleware=AuthMiddleware -PassContext
func (c *ProductController) CreateProductInCategory(ctx axon.RequestContext, categoryId uuid.UUID, req models.CreateProductRequest) (*models.Product, error) {
	// Mock implementation showing built-in UUID parser
	product := &models.Product{
		ID:          uuid.New(),
		CategoryID:  &categoryId,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
	}
	return product, nil
}

//axon::route PUT /products/{id:UUID}
func (c *ProductController) UpdateProduct(id uuid.UUID, req models.UpdateProductRequest) (*models.Product, error) {
	// Mock implementation using built-in UUID parser
	product := &models.Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
	}
	return product, nil
}

//axon::route DELETE /products/{id:UUID}
func (c *ProductController) DeleteProduct(id uuid.UUID) error {
	// Mock implementation using built-in UUID parser
	return nil
}