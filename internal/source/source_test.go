package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
	"github.com/RabbITCybErSeC/gofvml/internal/phys"
)

func TestFakeSource(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	fs := NewFakeSource(data, 0x1000)

	// Read all data.
	buf := make([]byte, len(data))
	n, err := fs.ReadAt(buf, 0x1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected %d bytes, got %d", len(data), n)
	}
	for i, want := range data {
		if buf[i] != want {
			t.Errorf("byte[%d] = %d, want %d", i, buf[i], want)
		}
	}

	// Read beyond data returns zeros.
	buf2 := make([]byte, 5)
	n, err = fs.ReadAt(buf2, 0x1000+uint64(len(data)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes, got %d", n)
	}
	for i := range buf2 {
		if buf2[i] != 0 {
			t.Errorf("expected zero byte[%d], got %d", i, buf2[i])
		}
	}

	// Close should work.
	if err := fs.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read after close should fail.
	_, err = fs.ReadAt(buf, 0x1000)
	if err == nil {
		t.Error("expected error after close")
	}
}

func TestFakeSourceAdapter(t *testing.T) {
	data := []byte{1, 2, 3}
	fake := NewFakeSource(data, 0)
	adapter := NewFakeSourceAdapter(fake)

	info := adapter.Info()
	if info.Name != "fake" {
		t.Errorf("Name = %q, want fake", info.Name)
	}

	ctx := context.Background()
	avail := adapter.Check(ctx)
	if !avail.Available {
		t.Error("expected fake source to be available")
	}

	reader, diag := adapter.Open(ctx)
	if diag != nil {
		t.Fatalf("unexpected diagnostic: %v", diag)
	}
	if reader == nil {
		t.Fatal("expected reader")
	}

	buf := make([]byte, 3)
	n, err := reader.ReadAt(buf, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 bytes, got %d", n)
	}
}

func TestRawSource(t *testing.T) {
	// Create a temporary file with known content.
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.raw")
	data := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer file.Close()

	raw := NewRawSource(file, tmpFile, "raw")

	// Read all data.
	buf := make([]byte, len(data))
	n, err := raw.ReadAt(buf, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected %d bytes, got %d", len(data), n)
	}
	for i, want := range data {
		if buf[i] != want {
			t.Errorf("byte[%d] = %d, want %d", i, buf[i], want)
		}
	}
}

func TestRawSourceAdapter(t *testing.T) {
	// Create a temporary file.
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.raw")
	if err := os.WriteFile(tmpFile, []byte{0, 1, 2, 3}, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	adapter := NewRawSourceAdapter(tmpFile, "raw")

	info := adapter.Info()
	if info.Name != "raw" {
		t.Errorf("Name = %q, want raw", info.Name)
	}
	if info.Path != tmpFile {
		t.Errorf("Path = %q, want %q", info.Path, tmpFile)
	}

	ctx := context.Background()
	avail := adapter.Check(ctx)
	if !avail.Available {
		t.Fatalf("expected source to be available, got: %s", avail.Reason)
	}

	reader, diag := adapter.Open(ctx)
	if diag != nil {
		t.Fatalf("unexpected diagnostic: %v", diag)
	}
	if reader == nil {
		t.Fatal("expected reader")
	}
	defer reader.Close()

	// Test reading.
	buf := make([]byte, 4)
	n, err := reader.ReadAt(buf, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 4 {
		t.Errorf("expected 4 bytes, got %d", n)
	}
}

func TestRawSourceAdapterNotFound(t *testing.T) {
	adapter := NewRawSourceAdapter("/nonexistent/path", "raw")

	ctx := context.Background()
	avail := adapter.Check(ctx)
	if avail.Available {
		t.Error("expected source to be unavailable")
	}
	if avail.Diagnostic == nil {
		t.Error("expected diagnostic for unavailable source")
	}
}

func TestReadBlock(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	fs := NewFakeSource(data, 0)

	block := phys.Block{Range: phys.Range{Start: 0, End: 3}}
	readBlock, err := ReadBlock(fs, block)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(readBlock.Data) != 3 {
		t.Errorf("expected 3 bytes, got %d", len(readBlock.Data))
	}
	for i := 0; i < 3; i++ {
		if readBlock.Data[i] != byte(i+1) {
			t.Errorf("byte[%d] = %d, want %d", i, readBlock.Data[i], i+1)
		}
	}
}

func TestReadBlocks(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6}
	fs := NewFakeSource(data, 0)

	blocks := []phys.Block{
		{Range: phys.Range{Start: 0, End: 2}},
		{Range: phys.Range{Start: 2, End: 4}},
		{Range: phys.Range{Start: 4, End: 6}},
	}

	results, errs := ReadBlocks(context.Background(), fs, blocks)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if len(errs) != 3 {
		t.Fatalf("expected 3 errors, got %d", len(errs))
	}

	for i, result := range results {
		if errs[i] != nil {
			t.Errorf("unexpected error for block %d: %v", i, errs[i])
			continue
		}
		if len(result.Data) != 2 {
			t.Errorf("expected 2 bytes for block %d, got %d", i, len(result.Data))
		}
	}
}

func TestReadBlocksCancellation(t *testing.T) {
	data := make([]byte, 100)
	fs := NewFakeSource(data, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	blocks := []phys.Block{
		{Range: phys.Range{Start: 0, End: 10}},
		{Range: phys.Range{Start: 10, End: 20}},
	}

	results, errs := ReadBlocks(ctx, fs, blocks)
	if len(results) != 1 {
		t.Fatalf("expected 1 result after cancellation, got %d", len(results))
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error after cancellation, got %d", len(errs))
	}
	if errs[0] != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", errs[0])
	}
}

func TestIsAllZeros(t *testing.T) {
	tests := []struct {
		data []byte
		want bool
	}{
		{[]byte{}, true},
		{[]byte{0, 0, 0}, true},
		{[]byte{0, 1, 0}, false},
		{[]byte{1}, false},
	}

	for _, tt := range tests {
		if got := IsAllZeros(tt.data); got != tt.want {
			t.Errorf("IsAllZeros(%v) = %v, want %v", tt.data, got, tt.want)
		}
	}
}

func TestDefaultSources(t *testing.T) {
	sources := DefaultSources()
	if len(sources) != 3 {
		t.Fatalf("expected 3 default sources, got %d", len(sources))
	}

	names := make([]string, len(sources))
	for i, s := range sources {
		names[i] = s.Info().Name
	}

	expected := []string{"crash", "kcore", "mem"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("source[%d].Name = %q, want %q", i, names[i], want)
		}
	}
}

func TestFindAvailableSource(t *testing.T) {
	// Test with fake source.
	fake := NewFakeSource([]byte{1, 2, 3}, 0)
	adapter := NewFakeSourceAdapter(fake)

	sources := []Source{adapter}
	ctx := context.Background()
	found, avail := FindAvailableSource(ctx, sources)
	if found == nil {
		t.Fatal("expected to find available source")
	}
	if !avail.Available {
		t.Error("expected availability to be true")
	}
	if found.Info().Name != "fake" {
		t.Errorf("Name = %q, want fake", found.Info().Name)
	}
}

func TestFindAvailableSourceNone(t *testing.T) {
	// Test with no sources.
	sources := []Source{}
	ctx := context.Background()
	found, avail := FindAvailableSource(ctx, sources)
	if found != nil {
		t.Error("expected no source found")
	}
	if avail.Available {
		t.Error("expected availability to be false")
	}
}

func TestOpenExplicit(t *testing.T) {
	fake := NewFakeSource([]byte{1, 2, 3}, 0)
	adapter := NewFakeSourceAdapter(fake)

	sources := []Source{adapter}
	ctx := context.Background()

	reader, diag := OpenExplicit(ctx, sources, "fake")
	if diag != nil {
		t.Fatalf("unexpected diagnostic: %v", diag)
	}
	if reader == nil {
		t.Fatal("expected reader")
	}
}

func TestOpenExplicitUnknown(t *testing.T) {
	sources := []Source{}
	ctx := context.Background()

	reader, diag := OpenExplicit(ctx, sources, "unknown")
	if reader != nil {
		t.Error("expected no reader")
	}
	if diag == nil {
		t.Fatal("expected diagnostic")
	}
	if diag.Category != diagnostic.CategorySource {
		t.Errorf("Category = %v, want source", diag.Category)
	}
}

func TestOpenExplicitUnavailable(t *testing.T) {
	// Use a raw source adapter pointing to a non-existent file.
	adapter := NewRawSourceAdapter("/nonexistent", "raw")
	sources := []Source{adapter}
	ctx := context.Background()

	reader, diag := OpenExplicit(ctx, sources, "raw")
	if reader != nil {
		t.Error("expected no reader")
	}
	if diag == nil {
		t.Fatal("expected diagnostic")
	}
}

func TestDevSourcesStub(t *testing.T) {
	// On non-Linux, these should be stubs that report unsupported platform.
	mem := NewDevMemSource()
	crash := NewDevCrashSource()
	kcore := NewProcKcoreSource()

	ctx := context.Background()

	for _, src := range []Source{mem, crash, kcore} {
		avail := src.Check(ctx)
		if avail.Available {
			t.Errorf("expected %s to be unavailable on this platform", src.Info().Name)
		}
		if avail.Diagnostic == nil {
			t.Errorf("expected diagnostic for unavailable %s", src.Info().Name)
		}

		reader, diag := src.Open(ctx)
		if reader != nil {
			t.Errorf("expected no reader for %s", src.Info().Name)
		}
		if diag == nil {
			t.Errorf("expected diagnostic for %s open", src.Info().Name)
		}
	}
}
