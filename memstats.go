package templar

import (
	"fmt"
	"io"
	"runtime"
	"strings"
	"time"
)

// MemSnapshot captures memory statistics at a point in time.
type MemSnapshot struct {
	// Name identifies this snapshot (e.g., "before-load", "after-render")
	Name string

	// Timestamp when the snapshot was taken
	Timestamp time.Time

	// Alloc is bytes of allocated heap objects.
	// This is the most useful metric for tracking "live" memory.
	Alloc uint64

	// TotalAlloc is cumulative bytes allocated (never decreases).
	// Useful for tracking allocation pressure.
	TotalAlloc uint64

	// HeapObjects is the number of allocated heap objects.
	HeapObjects uint64

	// HeapInuse is bytes in in-use spans.
	HeapInuse uint64

	// NumGC is the number of completed GC cycles.
	NumGC uint32

	// PauseTotalNs is cumulative nanoseconds in GC stop-the-world pauses.
	PauseTotalNs uint64
}

// MemStats collects memory snapshots for analysis.
type MemStats struct {
	snapshots []*MemSnapshot
}

// NewMemStats creates a new memory statistics collector.
func NewMemStats() *MemStats {
	return &MemStats{
		snapshots: make([]*MemSnapshot, 0),
	}
}

// Snapshot captures current memory statistics with the given name.
// Call this before and after operations you want to measure.
func (m *MemStats) Snapshot(name string) *MemSnapshot {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	snap := &MemSnapshot{
		Name:         name,
		Timestamp:    time.Now(),
		Alloc:        ms.Alloc,
		TotalAlloc:   ms.TotalAlloc,
		HeapObjects:  ms.HeapObjects,
		HeapInuse:    ms.HeapInuse,
		NumGC:        ms.NumGC,
		PauseTotalNs: ms.PauseTotalNs,
	}

	m.snapshots = append(m.snapshots, snap)
	return snap
}

// SnapshotWithGC forces a garbage collection before taking the snapshot.
// This gives a more accurate picture of "live" memory but is slower.
func (m *MemStats) SnapshotWithGC(name string) *MemSnapshot {
	runtime.GC()
	return m.Snapshot(name)
}

// Snapshots returns all captured snapshots.
func (m *MemStats) Snapshots() []*MemSnapshot {
	return m.snapshots
}

// Reset clears all captured snapshots.
func (m *MemStats) Reset() {
	m.snapshots = m.snapshots[:0]
}

// Delta calculates the difference between two named snapshots.
// Returns nil if either snapshot is not found.
func (m *MemStats) Delta(fromName, toName string) *MemDelta {
	var from, to *MemSnapshot
	for _, s := range m.snapshots {
		if s.Name == fromName {
			from = s
		}
		if s.Name == toName {
			to = s
		}
	}
	if from == nil || to == nil {
		return nil
	}
	return NewMemDelta(from, to)
}

// Report writes a formatted report of all snapshots to the writer.
func (m *MemStats) Report(w io.Writer) {
	if len(m.snapshots) == 0 {
		fmt.Fprintln(w, "No snapshots captured")
		return
	}

	// Header
	fmt.Fprintf(w, "%-20s | %12s | %12s | %12s | %8s | %12s\n",
		"Phase", "Alloc", "TotalAlloc", "HeapInuse", "Objects", "NumGC")
	fmt.Fprintln(w, strings.Repeat("-", 90))

	// Snapshots
	for _, s := range m.snapshots {
		fmt.Fprintf(w, "%-20s | %12s | %12s | %12s | %8d | %12d\n",
			truncate(s.Name, 20),
			formatBytes(s.Alloc),
			formatBytes(s.TotalAlloc),
			formatBytes(s.HeapInuse),
			s.HeapObjects,
			s.NumGC)
	}

	// Deltas between consecutive snapshots
	if len(m.snapshots) > 1 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Deltas:")
		fmt.Fprintf(w, "%-30s | %12s | %12s | %12s | %10s\n",
			"Transition", "ΔAlloc", "ΔTotalAlloc", "ΔObjects", "Duration")
		fmt.Fprintln(w, strings.Repeat("-", 85))

		for i := 1; i < len(m.snapshots); i++ {
			delta := NewMemDelta(m.snapshots[i-1], m.snapshots[i])
			fmt.Fprintf(w, "%-30s | %12s | %12s | %+10d | %10s\n",
				truncate(delta.FromName+" → "+delta.ToName, 30),
				formatBytesDelta(delta.AllocDelta),
				formatBytesDelta(delta.TotalAllocDelta),
				delta.HeapObjectsDelta,
				delta.Duration.Round(time.Microsecond))
		}
	}
}

// MemDelta represents the difference between two memory snapshots.
type MemDelta struct {
	FromName         string
	ToName           string
	Duration         time.Duration
	AllocDelta       int64
	TotalAllocDelta  int64
	HeapObjectsDelta int64
	HeapInuseDelta   int64
	NumGCDelta       int32
}

// NewMemDelta calculates the delta between two snapshots.
func NewMemDelta(from, to *MemSnapshot) *MemDelta {
	return &MemDelta{
		FromName:         from.Name,
		ToName:           to.Name,
		Duration:         to.Timestamp.Sub(from.Timestamp),
		AllocDelta:       int64(to.Alloc) - int64(from.Alloc),
		TotalAllocDelta:  int64(to.TotalAlloc) - int64(from.TotalAlloc),
		HeapObjectsDelta: int64(to.HeapObjects) - int64(from.HeapObjects),
		HeapInuseDelta:   int64(to.HeapInuse) - int64(from.HeapInuse),
		NumGCDelta:       int32(to.NumGC) - int32(from.NumGC),
	}
}

// String returns a human-readable summary of the delta.
func (d *MemDelta) String() string {
	return fmt.Sprintf("%s → %s: Alloc %s, TotalAlloc %s, Objects %+d, Duration %s",
		d.FromName, d.ToName,
		formatBytesDelta(d.AllocDelta),
		formatBytesDelta(d.TotalAllocDelta),
		d.HeapObjectsDelta,
		d.Duration.Round(time.Microsecond))
}

// formatBytes formats bytes in human-readable form.
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// formatBytesDelta formats a byte delta with +/- sign.
func formatBytesDelta(b int64) string {
	sign := "+"
	if b < 0 {
		sign = "-"
		b = -b
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%s%d B", sign, b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%s%.1f %cB", sign, float64(b)/float64(div), "KMGTPE"[exp])
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
