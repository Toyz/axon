package services

import "context"

// axon::core
// axon::interface
type CrawlerService struct {
	// Add any dependencies or configurations needed for the service
}

func (s *CrawlerService) Start(ctx context.Context) error {
	go func() {
		select {}
	}()
	return nil
}

func (s *CrawlerService) Stop(ctx context.Context) error {
	return nil
}
