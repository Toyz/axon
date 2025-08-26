package services

import (
	"context"
	"fmt"
	"time"

	"github.com/toyz/axon/examples/simple-app/internal/logging"
)

// axon::core -Init=Background
// axon::interface
type CrawlerService struct {
	//axon::inject
	logger *logging.AppLogger

	ticker *time.Ticker
}

func (s *CrawlerService) Start(ctx context.Context) error {
	fmt.Println("CrawlerService started")
	if s.ticker == nil {
		s.ticker = time.NewTicker(time.Second * 5)
	}

	for {
		<-s.ticker.C
		s.logger.Info("CrawlerService tick")
	}
}

func (s *CrawlerService) Stop(ctx context.Context) error {
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}

	s.logger.Info("CrawlerService stopped")
	return nil
}
