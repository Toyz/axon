package parsers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/toyz/axon/pkg/axon"
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
func ParseProductCode(c axon.RequestContext, paramValue string) (ProductCode, error) {
	code := ProductCode(strings.ToUpper(paramValue))
	if err := code.Validate(); err != nil {
		return "", fmt.Errorf("invalid product code '%s': %w", paramValue, err)
	}
	return code, nil
}