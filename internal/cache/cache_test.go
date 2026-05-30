package cache

import "testing"

func TestHitMiss(t *testing.T) {
	c := New[string](3)
	c.Put("a", "alpha")

	if v, ok := c.Get("a"); !ok || v != "alpha" {
		t.Errorf("Get(a) = %q, %v; want alpha, true", v, ok)
	}
	if _, ok := c.Get("missing"); ok {
		t.Error("Get(missing) should miss")
	}
}

func TestEvictionOrder(t *testing.T) {
	c := New[int](3)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)
	// Access "a" — now LRU order back→front is b, c, a
	c.Get("a")
	// Put "d" — should evict "b" (LRU)
	c.Put("d", 4)

	cases := []struct {
		key     string
		present bool
	}{
		{"b", false},
		{"a", true},
		{"c", true},
		{"d", true},
	}
	for _, tc := range cases {
		_, ok := c.Get(tc.key)
		if ok != tc.present {
			t.Errorf("Get(%q) present=%v, want %v", tc.key, ok, tc.present)
		}
	}
}

func TestUpdateExisting(t *testing.T) {
	c := New[string](2)
	c.Put("k", "v1")
	c.Put("k", "v2")

	if v, ok := c.Get("k"); !ok || v != "v2" {
		t.Errorf("Get(k) = %q, %v; want v2, true", v, ok)
	}
	// Updating must not grow the list.
	if c.ll.Len() != 1 {
		t.Errorf("list len = %d, want 1 after update", c.ll.Len())
	}
}

func TestUpdatePromotesToFront(t *testing.T) {
	c := New[int](2)
	c.Put("a", 1)
	c.Put("b", 2)
	// Update "a" — promotes it; "b" becomes LRU
	c.Put("a", 10)
	c.Put("c", 3) // evicts "b"

	if _, ok := c.Get("b"); ok {
		t.Error("b should be evicted after a was updated")
	}
	if v, ok := c.Get("a"); !ok || v != 10 {
		t.Errorf("Get(a) = %v, %v; want 10, true", v, ok)
	}
}

func TestDisabled(t *testing.T) {
	for _, cap := range []int{0, -1} {
		c := New[string](cap)
		c.Put("k", "v")
		if _, ok := c.Get("k"); ok {
			t.Errorf("capacity %d: disabled cache should always miss", cap)
		}
	}
}

func TestCapacityOne(t *testing.T) {
	c := New[int](1)
	c.Put("a", 1)
	c.Put("b", 2)

	if _, ok := c.Get("a"); ok {
		t.Error("a should be evicted by b")
	}
	if v, ok := c.Get("b"); !ok || v != 2 {
		t.Errorf("Get(b) = %v, %v; want 2, true", v, ok)
	}
}
