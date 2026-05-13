package source

import (
	"context"
	"fmt"
	"sync"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
)

// FakeSource is an in-memory source for testing acquisition without
// privileged devices.
type FakeSource struct {
	mu     sync.RWMutex
	data   map[uint64]byte
	closed bool
	name   string
	path   string
}

// NewFakeSource creates a FakeSource with the given data starting at offset.
func NewFakeSource(data []byte, startOffset uint64) *FakeSource {
	fs := &FakeSource{
		data: make(map[uint64]byte),
		name: "fake",
		path: "fake://memory",
	}
	for i, b := range data {
		fs.data[startOffset+uint64(i)] = b
	}
	return fs
}

// ReadAt reads bytes from the fake source at the given offset.
func (fs *FakeSource) ReadAt(p []byte, off uint64) (int, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if fs.closed {
		return 0, fmt.Errorf("source closed")
	}

	for i := range p {
		if b, ok := fs.data[off+uint64(i)]; ok {
			p[i] = b
		} else {
			p[i] = 0
		}
	}
	return len(p), nil
}

// Close marks the fake source as closed.
func (fs *FakeSource) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.closed = true
	return nil
}

// FakeSourceAdapter wraps a FakeSource as a Source.
type FakeSourceAdapter struct {
	fake *FakeSource
}

// NewFakeSourceAdapter creates a Source wrapper around a FakeSource.
func NewFakeSourceAdapter(fake *FakeSource) *FakeSourceAdapter {
	return &FakeSourceAdapter{fake: fake}
}

// Info returns static information about the fake source.
func (f *FakeSourceAdapter) Info() Info {
	return Info{
		Name:     "fake",
		Path:     f.fake.path,
		Priority: 0,
	}
}

// Check always reports the fake source as available.
func (f *FakeSourceAdapter) Check(ctx context.Context) Availability {
	return Availability{
		Available: true,
		Path:      f.fake.path,
		Reason:    "fake source always available",
	}
}

// Open returns the underlying FakeSource reader.
func (f *FakeSourceAdapter) Open(ctx context.Context) (Reader, *diagnostic.Diagnostic) {
	return f.fake, nil
}

// Ensure FakeSource implements Reader.
var _ Reader = (*FakeSource)(nil)

// Ensure FakeSourceAdapter implements Source.
var _ Source = (*FakeSourceAdapter)(nil)
