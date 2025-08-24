package parsers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// ProductCode represents a custom product code format: "PROD-12345"
type ProductCode string

// Validate checks if the product code has the correct format
func (pc ProductCode) Validate() error {
	s := string(pc)
	if !strings.HasPrefix(s, "PROD-") {
		return fmt.Errorf("product code must start with 'PROD-'")
	}
	if len(s) != 10 { // "PROD-" (5) + 5 digits = 10
		return fmt.Errorf("product code must be exactly 10 characters")
	}
	// Check that the part after "PROD-" is numeric
	numPart := s[5:]
	if _, err := strconv.Atoi(numPart); err != nil {
		return fmt.Errorf("product code must end with 5 digits")
	}
	return nil
}

// String returns the string representation
func (pc ProductCode) String() string {
	return string(pc)
}

//axon::route_parser ProductCode
func ParseProductCode(c echo.Context, paramValue string) (ProductCode, error) {
	code := ProductCode(strings.ToUpper(paramValue))
	if err := code.Validate(); err != nil {
		return "", fmt.Errorf("invalid product code '%s': %w", paramValue, err)
	}
	return code, nil
}

// DateRange represents a date range in format "2024-01-01_2024-12-31"
type DateRange struct {
	Start time.Time
	End   time.Time
}

//axon::route_parser DateRange
func ParseDateRange(c echo.Context, paramValue string) (DateRange, error) {
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