package middleware

import "github.com/labstack/echo/v4"

// axon::middleware GlobalMiddleware -Global -Priority=10
type GlobalMiddleware struct {
}

func (m *GlobalMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Pre-processing logic
		return next(c)
	}
}
