package parsers

import (
	"fmt"
	"strings"
	"time"

	"github.com/toyz/axon/pkg/axon"
)

// DateRange represents a date range in format "2024-01-01_2024-12-31"
type DateRange struct {
	Start time.Time
	End   time.Time
}

//axon::route_parser DateRange
func ParseDateRange(c axon.RequestContext, paramValue string) (DateRange, error) {
	parts := strings.Split(paramValue, "_")
	if len(parts) != 2 {
		return DateRange{}, fmt.Errorf("date range must be in format 'YYYY-MM-DD_YYYY-MM-DD'")
	}
	
	start, err := time.Parse("2006-01-02", parts[0])
	if err != nil {
		return DateRange{}, fmt.Errorf("invalid start date '%s': %w", parts[0], err)
	}
	
	end, err := time.Parse("2006-01-02", parts[1])
	if err != nil {
		return DateRange{}, fmt.Errorf("invalid end date '%s': %w", parts[1], err)
	}
	
	if end.Before(start) {
		return DateRange{}, fmt.Errorf("end date cannot be before start date")
	}
	
	return DateRange{Start: start, End: end}, nil
}