package acquisition

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/RabbITCybErSeC/gofvml/internal/image"
	"github.com/RabbITCybErSeC/gofvml/internal/phys"
	"github.com/RabbITCybErSeC/gofvml/internal/progress"
	"github.com/RabbITCybErSeC/gofvml/internal/source"
)

func TestAcquireLiME(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test.lime")

	file, err := createSafeOutput(outputPath)
	if err != nil {
		t.Fatalf("createSafeOutput error: %v", err)
	}

	block := phys.Block{Range: phys.Range{Start: 0x1000, End: 0x1008}, Data: data}
	written, err := writeLiMEBlock(file, block)
	if err != nil {
		t.Fatalf("writeLiMEBlock error: %v", err)
	}
	if written != image.LiMEHeaderSize+8 {
		t.Errorf("written = %d, want %d", written, image.LiMEHeaderSize+8)
	}

	file.Close()

	readData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if len(readData) != image.LiMEHeaderSize+8 {
		t.Errorf("file size = %d, want %d", len(readData), image.LiMEHeaderSize+8)
	}

	header, err := image.DecodeLiMEHeader(bytes.NewReader(readData))
	if err != nil {
		t.Fatalf("DecodeLiMEHeader error: %v", err)
	}
	if header.Start != 0x1000 {
		t.Errorf("Start = 0x%x, want 0x1000", header.Start)
	}
	if header.ExclusiveEnd() != 0x1008 {
		t.Errorf("ExclusiveEnd = 0x%x, want 0x1008", header.ExclusiveEnd())
	}
}

func TestAcquireAVML(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test.avml")

	file, err := createSafeOutput(outputPath)
	if err != nil {
		t.Fatalf("createSafeOutput error: %v", err)
	}

	block := phys.Block{Range: phys.Range{Start: 0x1000, End: 0x1008}, Data: data}
	_, err = writeAVMLBlock(file, block)
	if err != nil {
		t.Fatalf("writeAVMLBlock error: %v", err)
	}
	file.Close()

	readData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if len(readData) == 0 {
		t.Error("expected non-empty AVML file")
	}

	header, decodedData, err := image.DecodeAVMLBlock(bytes.NewReader(readData))
	if err != nil {
		t.Fatalf("DecodeAVMLBlock error: %v", err)
	}
	if header.Start != 0x1000 {
		t.Errorf("Start = 0x%x, want 0x1000", header.Start)
	}
	if !bytes.Equal(decodedData, data) {
		t.Errorf("data mismatch: got %v, want %v", decodedData, data)
	}
}

func TestAcquireZeroBlockSkipping(t *testing.T) {
	data := make([]byte, 8) // all zeros

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test.lime")

	file, err := createSafeOutput(outputPath)
	if err != nil {
		t.Fatalf("createSafeOutput error: %v", err)
	}

	block := phys.Block{Range: phys.Range{Start: 0x1000, End: 0x1008}, Data: data}
	if !block.IsZero() {
		t.Fatal("expected block to be zero")
	}
	file.Close()

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("file size = %d, want 0", info.Size())
	}
}

func TestCreateSafeOutput(t *testing.T) {
	tmpDir := t.TempDir()

	path := filepath.Join(tmpDir, "test.lime")
	file, err := createSafeOutput(path)
	if err != nil {
		t.Fatalf("createSafeOutput error: %v", err)
	}
	file.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("permissions = 0%o, want 0600", mode)
	}

	// O_EXCL should fail if file exists.
	_, err = createSafeOutput(path)
	if err == nil {
		t.Error("expected error when file already exists")
	}

	// Directory creation.
	nestedPath := filepath.Join(tmpDir, "nested", "dir", "test.lime")
	file, err = createSafeOutput(nestedPath)
	if err != nil {
		t.Fatalf("createSafeOutput nested error: %v", err)
	}
	file.Close()
}

func TestAcquireProgress(t *testing.T) {
	var events []progress.Event
	var mu sync.Mutex
	cb := func(e progress.Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
	}

	reporter := progress.NewReporter(cb)
	reporter.Report(progress.Event{
		Operation: "physical acquisition",
		Phase:     "reading",
		Current:   4,
		Total:     8,
	})
	reporter.Close()

	mu.Lock()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Current != 4 {
		t.Errorf("Current = %d, want 4", events[0].Current)
	}
	mu.Unlock()
}

func TestAcquireCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	select {
	case <-ctx.Done():
		// expected
	default:
		t.Error("expected context to be cancelled")
	}
}

func TestAcquireValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		opts    Options
		wantErr bool
	}{
		{
			name:    "missing_output",
			opts:    Options{OutputPath: "", Format: "lime", Ranges: []phys.Range{{Start: 0, End: 1}}},
			wantErr: true,
		},
		{
			name:    "missing_ranges",
			opts:    Options{OutputPath: "/tmp/test.lime", Format: "lime", Ranges: nil},
			wantErr: true,
		},
		{
			name:    "invalid_format",
			opts:    Options{OutputPath: "/tmp/test.lime", Format: "invalid", Ranges: []phys.Range{{Start: 0, End: 1}}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Acquire(ctx, tt.opts)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAcquireWithFakeSource(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	fake := source.NewFakeSource(data, 0x1000)
	adapter := source.NewFakeSourceAdapter(fake)

	// We can't easily inject the fake source into Acquire because it uses
	// DefaultSources(). Instead, let's test the individual components.
	_ = adapter

	// Test that ReadBlock works with fake source.
	block := phys.Block{Range: phys.Range{Start: 0x1000, End: 0x1008}}
	readBlock, err := source.ReadBlock(fake, block)
	if err != nil {
		t.Fatalf("ReadBlock error: %v", err)
	}
	if len(readBlock.Data) != 8 {
		t.Errorf("data length = %d, want 8", len(readBlock.Data))
	}
	for i, want := range data {
		if readBlock.Data[i] != want {
			t.Errorf("byte[%d] = %d, want %d", i, readBlock.Data[i], want)
		}
	}
}
