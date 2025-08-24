package services

import (
	"fmt"
	"time"
	"context"
)

// axon::core -Init=Background
// axon::interface
type CrawlerService struct {
	// Add any dependencies or configurations needed for the service
}

func (s *CrawlerService) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			fmt.Println("Crawling...")
			time.Sleep(1 * time.Second)
		}
	}
	return nil
}

func (s *CrawlerService) Stop(ctx context.Context) error {
	return nil
}
