package main

import (
	"fmt"

	"github.com/toyz/axon/pkg/axon"
)

func main() {
	converter := axon.NewRouteConverter()

	// Test wildcard conversion
	tests := []string{
		"/files/{*}",
		"/api/v1/{*}",
		"/static/{*}",
		"/users/{id:int}/files/{*}",
	}

	fmt.Println("Testing wildcard route conversion:")
	for _, test := range tests {
		echo := converter.AxonToEcho(test)
		fmt.Printf("Axon: %-30s -> Echo: %s\n", test, echo)

		// Test parameter extraction
		params := converter.ExtractParameterInfo(test)
		fmt.Printf("  Parameters: %v\n", params)

		// Test validation
		err := converter.ValidateAxonPath(test)
		if err != nil {
			fmt.Printf("  Validation error: %v\n", err)
		} else {
			fmt.Printf("  Validation: ✓ Valid\n")
		}
		fmt.Println()
	}

	// Test invalid wildcard routes
	fmt.Println("Testing invalid wildcard routes:")
	invalidTests := []string{
		"/files/{*}/more", // Wildcard not at end
		"/files/{*}/{*}",  // Multiple wildcards
	}

	for _, test := range invalidTests {
		err := converter.ValidateAxonPath(test)
		if err != nil {
			fmt.Printf("%-30s -> ✓ Correctly rejected: %v\n", test, err)
		} else {
			fmt.Printf("%-30s -> ✗ Should have been rejected\n", test)
		}
	}
}
