package presets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPresetManagerLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_presets_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "presets.json")

	// Load from non-existent file
	pm, err := LoadFrom(filePath)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if pm.Count() != 0 {
		t.Errorf("Expected 0 presets, got %d", pm.Count())
	}

	// Add valid presets
	if err := pm.Set(1, "/media/hdd/movies"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := pm.Set(2, "/media/hdd/series"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := pm.Set(99, "/media/downloads"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Invalid number < 1 or > 99
	if err := pm.Set(0, "/invalid"); err == nil {
		t.Errorf("Expected error for preset 0, got nil")
	}
	if err := pm.Set(100, "/invalid"); err == nil {
		t.Errorf("Expected error for preset 100, got nil")
	}

	// Empty path
	if err := pm.Set(5, "   "); err == nil {
		t.Errorf("Expected error for empty path, got nil")
	}

	// Test Resolve
	res1 := pm.Resolve("1")
	if res1 != "/media/hdd/movies" {
		t.Errorf("Expected /media/hdd/movies, got %s", res1)
	}

	resWithHash := pm.Resolve("#2")
	if resWithHash != "/media/hdd/series" {
		t.Errorf("Expected /media/hdd/series, got %s", resWithHash)
	}

	// Unknown preset number
	resUnknown := pm.Resolve("42")
	if resUnknown != "42" {
		t.Errorf("Expected 42, got %s", resUnknown)
	}

	// Regular path string
	resPath := pm.Resolve("/var/media/test")
	if resPath != "/var/media/test" {
		t.Errorf("Expected /var/media/test, got %s", resPath)
	}

	// Verify persistence by reloading from disk
	pmReloaded, err := LoadFrom(filePath)
	if err != nil {
		t.Fatalf("LoadFrom reloaded failed: %v", err)
	}
	if pmReloaded.Count() != 3 {
		t.Errorf("Expected 3 presets after reload, got %d", pmReloaded.Count())
	}

	list := pmReloaded.List()
	if len(list) != 3 {
		t.Fatalf("Expected list length 3, got %d", len(list))
	}
	if list[0].Number != 1 || list[1].Number != 2 || list[2].Number != 99 {
		t.Errorf("List order incorrect: %+v", list)
	}

	// Delete preset 2
	if err := pmReloaded.Delete(2); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if pmReloaded.Count() != 2 {
		t.Errorf("Expected 2 presets after delete, got %d", pmReloaded.Count())
	}
	if _, found := pmReloaded.Get(2); found {
		t.Errorf("Expected preset 2 to be deleted")
	}

	// Verify delete persisted
	pmAfterDelete, err := LoadFrom(filePath)
	if err != nil {
		t.Fatalf("LoadFrom after delete failed: %v", err)
	}
	if pmAfterDelete.Count() != 2 {
		t.Errorf("Expected 2 presets persisted, got %d", pmAfterDelete.Count())
	}
}

func TestParsePresetNumber(t *testing.T) {
	cases := []struct {
		input    string
		expected int
		valid    bool
	}{
		{"1", 1, true},
		{"99", 99, true},
		{"#1", 1, true},
		{" #42 ", 42, true},
		{"0", 0, false},
		{"100", 0, false},
		{"-5", 0, false},
		{"abc", 0, false},
		{"/path/1", 0, false},
		{"", 0, false},
	}

	for _, c := range cases {
		num, ok := ParsePresetNumber(c.input)
		if ok != c.valid || (ok && num != c.expected) {
			t.Errorf("ParsePresetNumber(%q) = (%d, %v), expected (%d, %v)", c.input, num, ok, c.expected, c.valid)
		}
	}
}
