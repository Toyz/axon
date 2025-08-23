package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

//axon::middleware AuthMiddleware
type AuthMiddleware struct {
}

// Handle implements the middleware logic for authentication
func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Check for Authorization header
		auth := c.Request().Header.Get("Authorization")
		if auth == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
		}
		
		// Simple token validation (in real app, validate JWT or similar)
		if !strings.HasPrefix(auth, "Bearer ") {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization format")
		}
		
		token := strings.TrimPrefix(auth, "Bearer ")
		if token != "valid-token" {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}
		
		// Set user context (in real app, decode from JWT)
		c.Set("user_id", 1)
		c.Set("user_name", "authenticated_user")
		
		return next(c)
	}
}