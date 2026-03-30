package templar

import (
	"testing"
)

// TestDefaultToolInfo verifies that DefaultToolInfo returns a fully populated
// ToolInfo with templar's standard naming conventions. This ensures the
// backward-compatible defaults are correct and no fields are accidentally empty.
func TestDefaultToolInfo(t *testing.T) {
	info := DefaultToolInfo()

	if info.Name != "templar" {
		t.Errorf("Expected Name 'templar', got '%s'", info.Name)
	}
	if len(info.ConfigNames) != 2 {
		t.Errorf("Expected 2 config names, got %d", len(info.ConfigNames))
	}
	if info.ConfigNames[0] != "templar.yaml" {
		t.Errorf("Expected first config name 'templar.yaml', got '%s'", info.ConfigNames[0])
	}
	if info.ConfigNames[1] != ".templar.yaml" {
		t.Errorf("Expected second config name '.templar.yaml', got '%s'", info.ConfigNames[1])
	}
	if info.VendorDir != "./templar_modules" {
		t.Errorf("Expected VendorDir './templar_modules', got '%s'", info.VendorDir)
	}
	if info.LockFile != "templar.lock" {
		t.Errorf("Expected LockFile 'templar.lock', got '%s'", info.LockFile)
	}
	if info.FetchCmd != "templar get" {
		t.Errorf("Expected FetchCmd 'templar get', got '%s'", info.FetchCmd)
	}
	if info.ProjectURL == "" {
		t.Error("Expected ProjectURL to be non-empty")
	}
}

// TestDefaultConstants verifies that the exported constants match the expected
// templar default values. These constants are used by CLI code to reference
// default file names without reconstructing a full ToolInfo.
func TestDefaultConstants(t *testing.T) {
	if DefaultVendorDir != "./templar_modules" {
		t.Errorf("Expected DefaultVendorDir './templar_modules', got '%s'", DefaultVendorDir)
	}
	if DefaultLockFile != "templar.lock" {
		t.Errorf("Expected DefaultLockFile 'templar.lock', got '%s'", DefaultLockFile)
	}
	if len(DefaultConfigNames) != 2 {
		t.Errorf("Expected 2 DefaultConfigNames, got %d", len(DefaultConfigNames))
	}
}
