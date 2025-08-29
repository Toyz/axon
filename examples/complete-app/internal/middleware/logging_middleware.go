package middleware

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
)

//axon::middleware LoggingMiddleware -Global
type LoggingMiddleware struct {
}

// Handle implements the middleware logic for request logging
func (m *LoggingMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		
		// Call the next handler
		err := next(c)
		
		// Log the request
		duration := time.Since(start)
		fmt.Printf("[%s] %s %s - %d - %v\n",
			start.Format("2006-01-02 15:04:05"),
			c.Request().Method,
			c.Request().URL.Path,
			c.Response().Status,
			duration,
		)
		
		return err
	}
}