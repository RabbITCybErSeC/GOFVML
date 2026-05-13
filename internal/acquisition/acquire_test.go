package acquisition

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
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

func TestAcquireAVMLSplitsLargeBlocks(t *testing.T) {
	data := bytes.Repeat([]byte{0x7b}, image.AVMLBlockSize+2)
	var output bytes.Buffer

	block := phys.Block{
		Range: phys.Range{Start: 0x1000, End: 0x1000 + uint64(len(data))},
		Data:  data,
	}
	if _, err := writeAVMLBlock(&output, block); err != nil {
		t.Fatalf("writeAVMLBlock error: %v", err)
	}

	r := bytes.NewReader(output.Bytes())
	header1, payload1, err := image.DecodeAVMLBlock(r)
	if err != nil {
		t.Fatalf("DecodeAVMLBlock first error: %v", err)
	}
	header2, payload2, err := image.DecodeAVMLBlock(r)
	if err != nil {
		t.Fatalf("DecodeAVMLBlock second error: %v", err)
	}
	if r.Len() != 0 {
		t.Fatalf("unexpected trailing bytes: %d", r.Len())
	}
	if header1.Start != 0x1000 || header1.RangeLen() != image.AVMLBlockSize {
		t.Fatalf("first block range = 0x%x len %d", header1.Start, header1.RangeLen())
	}
	if header2.Start != 0x1000+image.AVMLBlockSize || header2.RangeLen() != 2 {
		t.Fatalf("second block range = 0x%x len %d", header2.Start, header2.RangeLen())
	}
	if !bytes.Equal(append(payload1, payload2...), data) {
		t.Fatal("split payloads do not reconstruct original data")
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

	var output bufferWriteCloser
	result, err := Acquire(context.Background(), Options{
		OutputPath: "memory.lime",
		Format:     "lime",
		Ranges:     []phys.Range{{Start: 0x1000, End: 0x1008}},
		Sources:    []source.Source{adapter},
		Output:     &output,
	})
	if err != nil {
		t.Fatalf("Acquire error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.BlocksWritten != 1 {
		t.Fatalf("BlocksWritten = %d, want 1", result.BlocksWritten)
	}

	// Test that ReadBlock works with a fake source.
	fake = source.NewFakeSource(data, 0x1000)
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

func TestAcquireAllReadFailuresReturnError(t *testing.T) {
	var output bufferWriteCloser
	result, err := Acquire(context.Background(), Options{
		OutputPath: "memory.lime",
		Format:     "lime",
		Ranges:     []phys.Range{{Start: 0x1000, End: 0x1008}},
		Sources:    []source.Source{testSource{reader: failingReader{err: fmt.Errorf("read boom")}}},
		Output:     &output,
	})
	if err == nil {
		t.Fatal("expected acquisition error")
	}
	if result == nil {
		t.Fatal("expected partial result")
	}
	if result.Success {
		t.Error("expected unsuccessful result")
	}
	if result.BlocksWritten != 0 {
		t.Errorf("BlocksWritten = %d, want 0", result.BlocksWritten)
	}
	if len(result.Warnings) != 1 {
		t.Errorf("warnings = %d, want 1", len(result.Warnings))
	}
}

func TestAcquireFallsBackAfterAutoKcoreUnmapped(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	fake := source.NewFakeSource(data, 0x1000)
	var output bufferWriteCloser

	result, err := Acquire(context.Background(), Options{
		OutputPath: "memory.lime",
		Format:     "lime",
		Ranges:     []phys.Range{{Start: 0x1000, End: 0x1004}},
		Sources: []source.Source{
			testSource{name: source.SourceKcore, reader: notMappedReader{}},
			source.NewFakeSourceAdapter(fake),
		},
		Output: &output,
	})
	if err != nil {
		t.Fatalf("Acquire error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.SourceName != "fake" {
		t.Errorf("SourceName = %q, want fake", result.SourceName)
	}
	if result.BlocksWritten != 1 {
		t.Errorf("BlocksWritten = %d, want 1", result.BlocksWritten)
	}
}

func TestAcquireExplicitKcoreUnmappedDoesNotFallback(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	fake := source.NewFakeSource(data, 0x1000)
	var output bufferWriteCloser

	result, err := Acquire(context.Background(), Options{
		OutputPath: "memory.lime",
		Format:     "lime",
		SourceName: source.SourceKcore,
		Ranges:     []phys.Range{{Start: 0x1000, End: 0x1004}},
		Sources: []source.Source{
			testSource{name: source.SourceKcore, reader: notMappedReader{}},
			source.NewFakeSourceAdapter(fake),
		},
		Output: &output,
	})
	if err == nil {
		t.Fatal("expected acquisition error")
	}
	if result == nil {
		t.Fatal("expected partial result")
	}
	if result.Success {
		t.Error("expected unsuccessful result")
	}
	if result.BlocksWritten != 0 {
		t.Errorf("BlocksWritten = %d, want 0", result.BlocksWritten)
	}
}

func TestAcquireAllWriteFailuresReturnError(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	fake := source.NewFakeSource(data, 0x1000)
	result, err := Acquire(context.Background(), Options{
		OutputPath: "memory.lime",
		Format:     "lime",
		Ranges:     []phys.Range{{Start: 0x1000, End: 0x1004}},
		Sources:    []source.Source{source.NewFakeSourceAdapter(fake)},
		Output:     failingWriteCloser{err: fmt.Errorf("write boom")},
	})
	if err == nil {
		t.Fatal("expected acquisition error")
	}
	if result == nil {
		t.Fatal("expected partial result")
	}
	if result.Success {
		t.Error("expected unsuccessful result")
	}
	if result.BlocksWritten != 0 {
		t.Errorf("BlocksWritten = %d, want 0", result.BlocksWritten)
	}
	if len(result.Warnings) != 1 {
		t.Errorf("warnings = %d, want 1", len(result.Warnings))
	}
}

func TestAcquireAllZeroSkippedBlocksReturnError(t *testing.T) {
	data := make([]byte, 4)
	fake := source.NewFakeSource(data, 0x1000)
	var output bufferWriteCloser

	result, err := Acquire(context.Background(), Options{
		OutputPath:     "memory.lime",
		Format:         "lime",
		Ranges:         []phys.Range{{Start: 0x1000, End: 0x1004}},
		SkipZeroBlocks: true,
		Sources:        []source.Source{source.NewFakeSourceAdapter(fake)},
		Output:         &output,
	})
	if err == nil {
		t.Fatal("expected acquisition error")
	}
	if result == nil {
		t.Fatal("expected partial result")
	}
	if result.Success {
		t.Error("expected unsuccessful result")
	}
	if result.BlocksWritten != 0 {
		t.Errorf("BlocksWritten = %d, want 0", result.BlocksWritten)
	}
	if result.BlocksSkipped != 1 {
		t.Errorf("BlocksSkipped = %d, want 1", result.BlocksSkipped)
	}
	if output.Len() != 0 {
		t.Errorf("output length = %d, want 0", output.Len())
	}
}

type bufferWriteCloser struct {
	bytes.Buffer
}

func (b *bufferWriteCloser) Close() error {
	return nil
}

type failingWriteCloser struct {
	err error
}

func (f failingWriteCloser) Write([]byte) (int, error) {
	return 0, f.err
}

func (f failingWriteCloser) Close() error {
	return nil
}

type failingReader struct {
	err error
}

func (f failingReader) ReadAt([]byte, uint64) (int, error) {
	return 0, f.err
}

func (f failingReader) Close() error {
	return nil
}

type notMappedReader struct{}

func (n notMappedReader) ReadAt([]byte, uint64) (int, error) {
	return 0, source.NewNotMappedError(0x1000, source.PathProcKcore)
}

func (n notMappedReader) Close() error {
	return nil
}

type testSource struct {
	name   string
	reader source.Reader
}

func (t testSource) Info() source.Info {
	name := t.name
	if name == "" {
		name = "test"
	}
	return source.Info{Name: name, Path: "test://source"}
}

func (t testSource) Check(context.Context) source.Availability {
	return source.Availability{Available: true, Path: "test://source"}
}

func (t testSource) Open(context.Context) (source.Reader, *diagnostic.Diagnostic) {
	return t.reader, nil
}
