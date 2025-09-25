package middleware

import (
	"fmt"
	"time"

	"github.com/toyz/axon/pkg/axon"
)

//axon::middleware LoggingMiddleware -Global
type LoggingMiddleware struct {
}

// Handle implements the middleware logic for request logging
func (m *LoggingMiddleware) Handle(next axon.HandlerFunc) axon.HandlerFunc {
	return func(c axon.RequestContext) error {
		start := time.Now()

		// Call the next handler
		err := next(c)

		// Log the request
		duration := time.Since(start)
		fmt.Printf("[%s] %s %s - %d - %v\n",
			start.Format("2006-01-02 15:04:05"),
			c.Method(),
			c.Path(),
			c.Response().Status(),
			duration,
		)

		return err
	}
}