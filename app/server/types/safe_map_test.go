package types

import (
	"sync"
	"testing"
)

func TestSafeMapGetSet(t *testing.T) {
	sm := NewSafeMap[string]()

	// Get from empty map
	if got := sm.Get("missing"); got != "" {
		t.Errorf("Get from empty map = %q, want empty string", got)
	}

	// Set and Get
	sm.Set("key1", "value1")
	if got := sm.Get("key1"); got != "value1" {
		t.Errorf("Get after Set = %q, want %q", got, "value1")
	}

	// Overwrite
	sm.Set("key1", "updated")
	if got := sm.Get("key1"); got != "updated" {
		t.Errorf("Get after overwrite = %q, want %q", got, "updated")
	}
}

func TestSafeMapDelete(t *testing.T) {
	sm := NewSafeMap[int]()
	sm.Set("a", 1)
	sm.Set("b", 2)

	sm.Delete("a")
	if got := sm.Get("a"); got != 0 {
		t.Errorf("Get after Delete = %d, want 0", got)
	}
	if sm.Len() != 1 {
		t.Errorf("Len after Delete = %d, want 1", sm.Len())
	}
}

func TestSafeMapUpdate(t *testing.T) {
	sm := NewSafeMap[int]()
	sm.Set("counter", 0)

	// Update with a value-type: fn receives a copy, so Set afterwards
	// puts the original back — use pointer types for mutation.
	sm.Update("counter", func(v int) {
		// v is a copy; changes here don't propagate
	})
	if got := sm.Get("counter"); got != 0 {
		t.Errorf("Get after no-op Update = %d, want 0 (value unchanged)", got)
	}

	// Update missing key does nothing
	sm.Update("missing", func(v int) {
		v = 999
	})
	if sm.Len() != 1 {
		t.Errorf("Update missing key should not add it, Len = %d", sm.Len())
	}

	// With pointer types, mutation works
	smPtr := NewSafeMap[*int]()
	n := 41
	smPtr.Set("num", &n)
	smPtr.Update("num", func(v *int) {
		*v = 42
	})
	if got := *smPtr.Get("num"); got != 42 {
		t.Errorf("Get after pointer Update = %d, want 42", got)
	}
}

func TestSafeMapItems(t *testing.T) {
	sm := NewSafeMap[string]()
	sm.Set("x", "X")
	sm.Set("y", "Y")

	items := sm.Items()
	if len(items) != 2 {
		t.Errorf("Items len = %d, want 2", len(items))
	}
	if items["x"] != "X" || items["y"] != "Y" {
		t.Errorf("Items = %v, want {x:X, y:Y}", items)
	}

	// Items() returns a snapshot map copy.
	items["z"] = "Z"
	if sm.Get("z") != "" {
		t.Errorf("Modifying Items() return map should NOT affect SafeMap: Get('z') = %q", sm.Get("z"))
	}
}

func TestSafeMapKeys(t *testing.T) {
	sm := NewSafeMap[int]()
	sm.Set("one", 1)
	sm.Set("two", 2)

	keys := sm.Keys()
	if len(keys) != 2 {
		t.Errorf("Keys len = %d, want 2", len(keys))
	}

	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	if !keySet["one"] || !keySet["two"] {
		t.Errorf("Keys = %v, expected to contain 'one' and 'two'", keys)
	}
}

func TestSafeMapLen(t *testing.T) {
	sm := NewSafeMap[bool]()
	if sm.Len() != 0 {
		t.Errorf("empty Len = %d, want 0", sm.Len())
	}
	sm.Set("a", true)
	if sm.Len() != 1 {
		t.Errorf("Len after 1 insert = %d, want 1", sm.Len())
	}
	sm.Set("b", false)
	sm.Set("c", true)
	if sm.Len() != 3 {
		t.Errorf("Len after 3 inserts = %d, want 3", sm.Len())
	}
	sm.Delete("b")
	if sm.Len() != 2 {
		t.Errorf("Len after delete = %d, want 2", sm.Len())
	}
}

func TestSafeMapConcurrent(t *testing.T) {
	sm := NewSafeMap[int]()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sm.Set(string(rune('a'+n%26)), n)
			sm.Get("any")
			sm.Len()
			sm.Update("a", func(v int) { v++ })
		}(i)
	}

	wg.Wait()
	// Should not panic and should have at most 26 entries
	if sm.Len() > 26 {
		t.Errorf("Len = %d, want at most 26", sm.Len())
	}
}

func TestSafeMapCustomType(t *testing.T) {
	type person struct {
		name string
		age  int
	}
	sm := NewSafeMap[person]()
	sm.Set("bob", person{"Bob", 30})
	sm.Set("alice", person{"Alice", 25})

	if got := sm.Get("bob"); got.name != "Bob" || got.age != 30 {
		t.Errorf("Get bob = %+v, want {Bob 30}", got)
	}
}
