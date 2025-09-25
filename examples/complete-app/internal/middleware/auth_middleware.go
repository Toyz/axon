package middleware

import (
	"net/http"
	"strings"

	"github.com/toyz/axon/examples/complete-app/internal/config"
	"github.com/toyz/axon/examples/complete-app/internal/services"
	"github.com/toyz/axon/pkg/axon"
)

// axon::middleware AuthMiddleware
type AuthMiddleware struct {
	// axon::inject
	sessionFactory func() *services.SessionService

	//axon::inject
	Config *config.Config
}

// Handle implements the middleware logic for authentication
func (m *AuthMiddleware) Handle(next axon.HandlerFunc) axon.HandlerFunc {
	return func(c axon.RequestContext) error {
		// Check for Authorization header
		auth := c.Request().Header("Authorization")
		if auth == "" {
			return axon.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
		}

		// Simple token validation (in real app, validate JWT or similar)
		if !strings.HasPrefix(auth, "Bearer ") {
			return axon.NewHTTPError(http.StatusUnauthorized, "invalid authorization format")
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token != "valid-token" {
			return axon.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}

		// Set user context (in real app, decode from JWT)
		c.Set("user_id", 1)
		c.Set("user_name", "authenticated_user")

		return next(c)
	}
}
