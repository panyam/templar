package templar

import (
	"bytes"
	"strings"
	"testing"
)

func TestMemStats_Snapshot(t *testing.T) {
	stats := NewMemStats()

	snap1 := stats.Snapshot("initial")
	if snap1.Name != "initial" {
		t.Errorf("Expected name 'initial', got '%s'", snap1.Name)
	}
	if snap1.Alloc == 0 {
		t.Error("Expected non-zero Alloc")
	}
	if snap1.Timestamp.IsZero() {
		t.Error("Expected non-zero Timestamp")
	}

	// Allocate some memory
	data := make([]byte, 1024*1024) // 1MB
	_ = data

	snap2 := stats.Snapshot("after-alloc")

	if len(stats.Snapshots()) != 2 {
		t.Errorf("Expected 2 snapshots, got %d", len(stats.Snapshots()))
	}

	// TotalAlloc should have increased
	if snap2.TotalAlloc <= snap1.TotalAlloc {
		t.Error("Expected TotalAlloc to increase after allocation")
	}
}

func TestMemStats_Delta(t *testing.T) {
	stats := NewMemStats()

	stats.Snapshot("start")

	// Allocate some memory
	data := make([]byte, 1024*1024) // 1MB
	_ = data

	stats.Snapshot("end")

	delta := stats.Delta("start", "end")
	if delta == nil {
		t.Fatal("Expected delta, got nil")
	}

	if delta.FromName != "start" || delta.ToName != "end" {
		t.Errorf("Unexpected delta names: %s -> %s", delta.FromName, delta.ToName)
	}

	// TotalAlloc delta should be at least 1MB
	if delta.TotalAllocDelta < 1024*1024 {
		t.Errorf("Expected TotalAllocDelta >= 1MB, got %d", delta.TotalAllocDelta)
	}

	if delta.Duration <= 0 {
		t.Error("Expected positive duration")
	}
}

func TestMemStats_DeltaNotFound(t *testing.T) {
	stats := NewMemStats()
	stats.Snapshot("exists")

	delta := stats.Delta("exists", "missing")
	if delta != nil {
		t.Error("Expected nil delta for missing snapshot")
	}
}

func TestMemStats_Report(t *testing.T) {
	stats := NewMemStats()

	stats.Snapshot("phase1")
	stats.Snapshot("phase2")

	var buf bytes.Buffer
	stats.Report(&buf)

	output := buf.String()

	// Check that report contains expected elements
	if !strings.Contains(output, "Phase") {
		t.Error("Report should contain 'Phase' header")
	}
	if !strings.Contains(output, "phase1") {
		t.Error("Report should contain 'phase1'")
	}
	if !strings.Contains(output, "phase2") {
		t.Error("Report should contain 'phase2'")
	}
	if !strings.Contains(output, "Deltas:") {
		t.Error("Report should contain 'Deltas:' section")
	}
}

func TestMemStats_Reset(t *testing.T) {
	stats := NewMemStats()

	stats.Snapshot("one")
	stats.Snapshot("two")

	if len(stats.Snapshots()) != 2 {
		t.Fatalf("Expected 2 snapshots, got %d", len(stats.Snapshots()))
	}

	stats.Reset()

	if len(stats.Snapshots()) != 0 {
		t.Errorf("Expected 0 snapshots after reset, got %d", len(stats.Snapshots()))
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tc := range tests {
		result := formatBytes(tc.input)
		if result != tc.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestFormatBytesDelta(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "+0 B"},
		{100, "+100 B"},
		{-100, "-100 B"},
		{1024, "+1.0 KB"},
		{-1048576, "-1.0 MB"},
	}

	for _, tc := range tests {
		result := formatBytesDelta(tc.input)
		if result != tc.expected {
			t.Errorf("formatBytesDelta(%d) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestMemDelta_String(t *testing.T) {
	stats := NewMemStats()
	stats.Snapshot("a")
	stats.Snapshot("b")

	delta := stats.Delta("a", "b")
	str := delta.String()

	if !strings.Contains(str, "a → b") {
		t.Error("Delta string should contain transition names")
	}
	if !strings.Contains(str, "Alloc") {
		t.Error("Delta string should contain Alloc")
	}
}
