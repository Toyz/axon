package utils

import (
	"errors"
	"testing"
)

func TestRegistry_BasicOperations(t *testing.T) {
	registry := NewRegistry[string, int]()

	// Test initial state
	if registry.Size() != 0 {
		t.Errorf("expected empty registry, got size %d", registry.Size())
	}

	// Test Register and Get
	err := registry.Register("key1", 42)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	value, exists := registry.Get("key1")
	if !exists {
		t.Error("expected key1 to exist")
	}
	if value != 42 {
		t.Errorf("expected value 42, got %d", value)
	}

	// Test Has
	if !registry.Has("key1") {
		t.Error("expected Has to return true for key1")
	}
	if registry.Has("nonexistent") {
		t.Error("expected Has to return false for nonexistent key")
	}

	// Test Size
	if registry.Size() != 1 {
		t.Errorf("expected size 1, got %d", registry.Size())
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry[string, string]()

	// Add some items
	registry.Register("a", "value_a")
	registry.Register("b", "value_b")
	registry.Register("c", "value_c")

	keys := registry.List()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}

	// Check that all keys are present (order doesn't matter)
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}

	expectedKeys := []string{"a", "b", "c"}
	for _, expected := range expectedKeys {
		if !keyMap[expected] {
			t.Errorf("expected key %s not found in list", expected)
		}
	}
}

func TestRegistry_GetAll(t *testing.T) {
	registry := NewRegistry[string, int]()

	// Add some items
	registry.Register("one", 1)
	registry.Register("two", 2)
	registry.Register("three", 3)

	all := registry.GetAll()
	if len(all) != 3 {
		t.Errorf("expected 3 items, got %d", len(all))
	}

	// Verify all items are present
	expected := map[string]int{"one": 1, "two": 2, "three": 3}
	for key, expectedValue := range expected {
		if value, exists := all[key]; !exists {
			t.Errorf("expected key %s not found", key)
		} else if value != expectedValue {
			t.Errorf("expected value %d for key %s, got %d", expectedValue, key, value)
		}
	}

	// Verify it's a copy (modifying returned map shouldn't affect registry)
	all["four"] = 4
	if registry.Has("four") {
		t.Error("modifying returned map should not affect registry")
	}
}

func TestRegistry_Clear(t *testing.T) {
	registry := NewRegistry[string, int]()

	// Add some items
	registry.Register("key1", 1)
	registry.Register("key2", 2)

	if registry.Size() != 2 {
		t.Errorf("expected size 2 before clear, got %d", registry.Size())
	}

	// Clear registry
	registry.Clear()

	if registry.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", registry.Size())
	}

	if registry.Has("key1") {
		t.Error("expected key1 to be removed after clear")
	}
}

func TestRegistry_ClearWithReset(t *testing.T) {
	registry := NewRegistry[string, int]()

	// Add some items
	registry.Register("key1", 1)
	registry.Register("key2", 2)

	// Clear with reset
	initialItems := map[string]int{"reset1": 10, "reset2": 20}
	registry.ClearWithReset(initialItems)

	if registry.Size() != 2 {
		t.Errorf("expected size 2 after reset, got %d", registry.Size())
	}

	// Check old items are gone
	if registry.Has("key1") {
		t.Error("expected key1 to be removed after reset")
	}

	// Check new items are present
	if value, exists := registry.Get("reset1"); !exists || value != 10 {
		t.Errorf("expected reset1 with value 10, got exists=%v, value=%d", exists, value)
	}
}

func TestRegistry_RegisterWithValidator(t *testing.T) {
	registry := NewRegistry[string, int]()

	// Test successful validation
	validator := func(key string, value int, existing map[string]int) error {
		if value < 0 {
			return errors.New("value must be non-negative")
		}
		if _, exists := existing[key]; exists {
			return errors.New("key already exists")
		}
		return nil
	}

	err := registry.RegisterWithValidator("key1", 42, validator)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test validation failure (negative value)
	err = registry.RegisterWithValidator("key2", -1, validator)
	if err == nil {
		t.Error("expected validation error for negative value")
	}

	// Test validation failure (duplicate key)
	err = registry.RegisterWithValidator("key1", 100, validator)
	if err == nil {
		t.Error("expected validation error for duplicate key")
	}

	// Verify only valid item was registered
	if registry.Size() != 1 {
		t.Errorf("expected size 1, got %d", registry.Size())
	}
}

func TestRegistry_ForEach(t *testing.T) {
	registry := NewRegistry[string, int]()

	// Add some items
	registry.Register("a", 1)
	registry.Register("b", 2)
	registry.Register("c", 3)

	// Test ForEach
	sum := 0
	registry.ForEach(func(key string, value int) {
		sum += value
	})

	expectedSum := 1 + 2 + 3
	if sum != expectedSum {
		t.Errorf("expected sum %d, got %d", expectedSum, sum)
	}
}

func TestRegistry_Filter(t *testing.T) {
	registry := NewRegistry[string, int]()

	// Add some items
	registry.Register("a", 1)
	registry.Register("b", 2)
	registry.Register("c", 3)
	registry.Register("d", 4)

	// Filter even values
	evenValues := registry.Filter(func(key string, value int) bool {
		return value%2 == 0
	})

	if len(evenValues) != 2 {
		t.Errorf("expected 2 even values, got %d", len(evenValues))
	}

	// Check that correct values are present
	if value, exists := evenValues["b"]; !exists || value != 2 {
		t.Errorf("expected b=2 in filtered results")
	}
	if value, exists := evenValues["d"]; !exists || value != 4 {
		t.Errorf("expected d=4 in filtered results")
	}
}

func TestRegistry_ThreadSafety(t *testing.T) {
	registry := NewRegistry[int, string]()

	// Test concurrent access
	done := make(chan bool, 2)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			registry.Register(i, "value")
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			registry.Get(i)
			registry.Has(i)
			registry.List()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify final state
	if registry.Size() != 100 {
		t.Errorf("expected size 100, got %d", registry.Size())
	}
}

func TestRegistry_DifferentTypes(t *testing.T) {
	// Test with different key/value types
	intRegistry := NewRegistry[int, string]()
	intRegistry.Register(1, "one")
	intRegistry.Register(2, "two")

	if value, exists := intRegistry.Get(1); !exists || value != "one" {
		t.Errorf("int key registry failed")
	}

	// Test with struct types
	type TestStruct struct {
		Name string
		ID   int
	}

	structRegistry := NewRegistry[string, TestStruct]()
	testItem := TestStruct{Name: "test", ID: 42}
	structRegistry.Register("item1", testItem)

	if value, exists := structRegistry.Get("item1"); !exists || value.Name != "test" || value.ID != 42 {
		t.Errorf("struct value registry failed")
	}
}