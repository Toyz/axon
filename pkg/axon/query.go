package axon

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

// QueryMap represents URL query parameters with convenient access methods
type QueryMap struct {
	values url.Values
}

// NewQueryMap creates a QueryMap from Echo context
func NewQueryMap(c echo.Context) QueryMap {
	return QueryMap{
		values: c.QueryParams(),
	}
}

// Get returns the first value for the given key, or empty string if not found
func (q QueryMap) Get(key string) string {
	return q.values.Get(key)
}

// GetDefault returns the first value for the given key, or the default value if not found
func (q QueryMap) GetDefault(key, defaultValue string) string {
	if value := q.values.Get(key); value != "" {
		return value
	}
	return defaultValue
}

// GetInt returns the first value for the given key as an integer, or 0 if not found/invalid
func (q QueryMap) GetInt(key string) int {
	if value := q.values.Get(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return 0
}

// GetIntDefault returns the first value for the given key as an integer, or the default if not found/invalid
func (q QueryMap) GetIntDefault(key string, defaultValue int) int {
	if value := q.values.Get(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// GetBool returns the first value for the given key as a boolean
// Accepts: "true", "1", "yes", "on" (case insensitive) as true
func (q QueryMap) GetBool(key string) bool {
	value := strings.ToLower(q.values.Get(key))
	return value == "true" || value == "1" || value == "yes" || value == "on"
}

// GetAll returns all values for the given key
func (q QueryMap) GetAll(key string) []string {
	return q.values[key]
}

// Has returns true if the key exists in the query parameters
func (q QueryMap) Has(key string) bool {
	_, exists := q.values[key]
	return exists
}

// Keys returns all query parameter keys
func (q QueryMap) Keys() []string {
	keys := make([]string, 0, len(q.values))
	for key := range q.values {
		keys = append(keys, key)
	}
	return keys
}

// ToMap returns the underlying url.Values as a map[string][]string
func (q QueryMap) ToMap() map[string][]string {
	return map[string][]string(q.values)
}