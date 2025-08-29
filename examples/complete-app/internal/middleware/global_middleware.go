package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/toyz/axon/examples/complete-app/internal/logging"
)

// axon::middleware GlobalMiddleware -Global -Priority=10
type GlobalMiddleware struct {
	// axon::inject
	logger *logging.AppLogger
}

func (m *GlobalMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Pre-processing logic
		return next(c)
	}
}
