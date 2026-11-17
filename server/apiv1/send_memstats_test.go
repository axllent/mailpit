package apiv1

import "runtime"

// memStats is what the memory tests of this package read.
type memStats struct{ heapInUse uint64 }

func (m *memStats) read() {
	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	m.heapInUse = s.HeapInuse
}

func (m *memStats) gc() { runtime.GC() }
