package middleware

import (
	"github.com/toyz/axon/examples/complete-app/internal/logging"
	"github.com/toyz/axon/pkg/axon"
)

// axon::middleware GlobalMiddleware -Global -Priority=10
type GlobalMiddleware struct {
	// axon::inject
	logger *logging.AppLogger
}

func (m *GlobalMiddleware) Handle(next axon.HandlerFunc) axon.HandlerFunc {
	return func(c axon.RequestContext) error {
		// Pre-processing logic
		m.logger.Info("GlobalMiddleware: Pre-processing logic")

		return next(c)
	}
}
