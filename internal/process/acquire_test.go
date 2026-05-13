package process

import (
	"bytes"
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/RabbITCybErSeC/gofvml/internal/procfs"
	"github.com/RabbITCybErSeC/gofvml/internal/progress"
	"github.com/RabbITCybErSeC/gofvml/internal/testhelpers"
)

func skipIfNotLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping: /proc not available on " + runtime.GOOS)
	}
}

func TestAcquireSelf(t *testing.T) {
	skipIfNotLinux(t)
	pid := os.Getpid()

	// Use a small chunk size to exercise chunking logic.
	opts := Options{
		PID:       pid,
		Filter:    DefaultFilter(),
		ChunkSize: 4096,
		Strict:    false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := Acquire(ctx, opts)
	if err != nil {
		t.Fatalf("Acquire(self) failed: %v", err)
	}

	if result.PID != pid {
		t.Errorf("result.PID = %d, want %d", result.PID, pid)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if len(result.Mappings) == 0 {
		t.Fatal("expected at least one mapping")
	}
	if result.BytesRead == 0 {
		t.Error("expected some bytes read")
	}

	// Verify we read from at least one mapping.
	hasRead := false
	for _, mr := range result.Mappings {
		if mr.BytesRead > 0 {
			hasRead = true
		}
		if len(mr.Events) == 0 {
			t.Errorf("mapping %x-%x has no events", mr.Mapping.Start, mr.Mapping.End)
		}
	}
	if !hasRead {
		t.Error("expected at least one mapping with bytes read")
	}
}

func TestAcquireSelfWithAddressFilter(t *testing.T) {
	skipIfNotLinux(t)
	pid := os.Getpid()

	// Read our own maps to find a specific readable mapping.
	maps, err := procfs.ReadMaps(pid)
	if err != nil {
		t.Skipf("cannot read self maps: %v", err)
	}

	var target procfs.Mapping
	for _, m := range maps {
		if m.IsReadable() && m.Len() >= 4096 {
			target = m
			break
		}
	}
	if target.Start == 0 {
		t.Skip("no suitable readable mapping found")
	}

	opts := Options{
		PID: pid,
		Filter: Filter{
			RequireReadable: true,
			MinAddress:      target.Start,
			MaxAddress:      target.Start + 4096,
		},
		ChunkSize: 4096,
		Strict:    false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := Acquire(ctx, opts)
	if err != nil {
		t.Fatalf("Acquire(self, filtered) failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(result.Mappings))
	}
	if result.Mappings[0].Mapping.Start != target.Start {
		t.Errorf("mapping start = %x, want %x", result.Mappings[0].Mapping.Start, target.Start)
	}
}

func TestAcquireCancellation(t *testing.T) {
	skipIfNotLinux(t)
	pid := os.Getpid()

	ctx := testhelpers.CancelImmediately(t)

	opts := Options{
		PID:       pid,
		Filter:    DefaultFilter(),
		ChunkSize: 4096,
	}

	_, err := Acquire(ctx, opts)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestAcquireInvalidPID(t *testing.T) {
	opts := Options{
		PID:       99999999,
		Filter:    DefaultFilter(),
		ChunkSize: 4096,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Acquire(ctx, opts)
	if err == nil {
		t.Fatal("expected error for invalid PID")
	}
}

func TestAcquireNonStrictKeepsSuccessfulBlocksWhenLaterMappingFails(t *testing.T) {
	readErr := errors.New("input/output error")
	proc := fakeProcessFS{
		maps: []procfs.Mapping{
			{Start: 0x1000, End: 0x1004, Perms: "r--p", Pathname: "[heap]"},
			{Start: 0x2000, End: 0x2004, Perms: "r--p", Pathname: "[stack]"},
		},
		mem: &scriptedMem{
			reads: map[int64]scriptedRead{
				0x1000: {data: []byte("MARK")},
				0x2000: {err: readErr},
			},
		},
	}

	result, err := acquireWithProcFS(context.Background(), Options{
		PID:       1706,
		Filter:    DefaultFilter(),
		ChunkSize: 4,
		Strict:    false,
	}, proc)
	if err != nil {
		t.Fatalf("Acquire returned error in non-strict mode: %v", err)
	}
	if !result.Success {
		t.Fatal("expected non-strict acquisition to report success after useful reads")
	}
	if result.BytesRead != 4 {
		t.Fatalf("BytesRead = %d, want 4", result.BytesRead)
	}
	if len(result.Mappings) != 2 {
		t.Fatalf("Mappings = %d, want 2", len(result.Mappings))
	}
	if got := result.Mappings[0].Blocks; len(got) != 1 || !bytes.Equal(got[0].Data, []byte("MARK")) {
		t.Fatalf("first mapping blocks = %+v, want MARK payload", got)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("Warnings = %d, want 1", len(result.Warnings))
	}
	warning := result.Warnings[0]
	if warning.Target != "pid=1706 mapping=2000-2004" {
		t.Fatalf("warning target = %q, want failed mapping target", warning.Target)
	}
	if warning.Cause == nil || !strings.Contains(warning.Cause.Error(), "input/output error") {
		t.Fatalf("warning cause = %v, want input/output error", warning.Cause)
	}
}

func TestAcquireStrictFailsOnFirstMappingError(t *testing.T) {
	readErr := errors.New("input/output error")
	proc := fakeProcessFS{
		maps: []procfs.Mapping{
			{Start: 0x1000, End: 0x1004, Perms: "r--p", Pathname: "[heap]"},
			{Start: 0x2000, End: 0x2004, Perms: "r--p", Pathname: "[stack]"},
			{Start: 0x3000, End: 0x3004, Perms: "r--p", Pathname: "[vdso]"},
		},
		mem: &scriptedMem{
			reads: map[int64]scriptedRead{
				0x1000: {data: []byte("GOOD")},
				0x2000: {err: readErr},
				0x3000: {data: []byte("LATE")},
			},
		},
	}

	result, err := acquireWithProcFS(context.Background(), Options{
		PID:       1706,
		Filter:    DefaultFilter(),
		ChunkSize: 4,
		Strict:    true,
	}, proc)
	if err == nil {
		t.Fatal("expected strict acquisition to return the mapping read error")
	}
	if !strings.Contains(err.Error(), "read mapping chunk at 0x2000") {
		t.Fatalf("error = %v, want failed mapping address", err)
	}
	if result == nil {
		t.Fatal("expected partial result in strict mode")
	}
	if result.Success {
		t.Fatal("expected strict acquisition result to be unsuccessful")
	}
	if result.BytesRead != 4 {
		t.Fatalf("BytesRead = %d, want successful bytes before failure", result.BytesRead)
	}
	if len(result.Mappings) != 2 {
		t.Fatalf("Mappings = %d, want acquisition to stop at failed mapping", len(result.Mappings))
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("Warnings = %d, want strict mode to return error without warning", len(result.Warnings))
	}
	if _, ok := proc.mem.readOffsets[0x3000]; ok {
		t.Fatal("strict acquisition read a mapping after the first failure")
	}
}

func TestAcquireProgressEvents(t *testing.T) {
	skipIfNotLinux(t)
	pid := os.Getpid()

	var events []progress.Event
	opts := Options{
		PID:    pid,
		Filter: DefaultFilter(),
		Progress: func(e progress.Event) {
			events = append(events, e)
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := Acquire(ctx, opts)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	if len(events) == 0 {
		t.Error("expected progress events")
	}
	for _, e := range events {
		if e.Operation != "process acquisition" {
			t.Errorf("unexpected operation: %q", e.Operation)
		}
		if e.Phase != "reading" {
			t.Errorf("unexpected phase: %q", e.Phase)
		}
	}
}

func TestReadEventHelpers(t *testing.T) {
	e := ReadEvent{Requested: 4096, Read: 4096}
	if e.IsError() {
		t.Error("expected not error")
	}
	if e.IsShortRead() {
		t.Error("expected not short read")
	}

	e2 := ReadEvent{Requested: 4096, Read: 2048}
	if e2.IsError() {
		t.Error("expected not error")
	}
	if !e2.IsShortRead() {
		t.Error("expected short read")
	}

	e3 := ReadEvent{Requested: 4096, Read: 0, Err: os.ErrNotExist}
	if !e3.IsError() {
		t.Error("expected error")
	}
	if e3.IsShortRead() {
		t.Error("expected not short read when error")
	}
}

func TestReadMappingCapturesPayloadBlocks(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/mem"
	data := []byte("abcdefghijkl")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	mapping := procfs.Mapping{Start: 0, End: uint64(len(data)), Perms: "r--p"}
	var mr MappingResult
	err = readMapping(context.Background(), f, mapping, 5, &mr)
	if err != nil {
		t.Fatalf("readMapping error: %v", err)
	}

	if got, want := len(mr.Blocks), 3; got != want {
		t.Fatalf("blocks = %d, want %d", got, want)
	}
	if got := append(append([]byte{}, mr.Blocks[0].Data...), append(mr.Blocks[1].Data, mr.Blocks[2].Data...)...); !bytes.Equal(got, data) {
		t.Fatalf("payload blocks reconstruct %q, want %q", got, data)
	}
	if mr.Blocks[0].VirtualAddress != 0 || mr.Blocks[1].VirtualAddress != 5 || mr.Blocks[2].VirtualAddress != 10 {
		t.Fatalf("unexpected block virtual addresses: %#v", mr.Blocks)
	}
}

func TestReadMappingPreservesPartialPayloadWhenReadReturnsBytesAndError(t *testing.T) {
	mem := &scriptedMem{
		reads: map[int64]scriptedRead{
			0x4000: {data: []byte("PART"), err: errors.New("input/output error")},
		},
	}
	mapping := procfs.Mapping{Start: 0x4000, End: 0x4008, Perms: "r--p"}
	var mr MappingResult

	err := readMapping(context.Background(), mem, mapping, 8, &mr)
	if err == nil {
		t.Fatal("expected readMapping error")
	}
	if mr.BytesRead != 4 {
		t.Fatalf("BytesRead = %d, want partial bytes", mr.BytesRead)
	}
	if len(mr.Blocks) != 1 {
		t.Fatalf("Blocks = %d, want one partial block", len(mr.Blocks))
	}
	if got := mr.Blocks[0].Data; !bytes.Equal(got, []byte("PART")) {
		t.Fatalf("partial payload = %q, want PART", got)
	}
	if mr.Blocks[0].Status != StatusError {
		t.Fatalf("partial block status = %d, want StatusError", mr.Blocks[0].Status)
	}
}

func TestReadMappingReturnsErrorOnFailedChunk(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/short-mem"
	if err := os.WriteFile(path, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	mapping := procfs.Mapping{Start: 0, End: 8, Perms: "r--p"}
	var mr MappingResult
	err = readMapping(context.Background(), f, mapping, 8, &mr)
	if err == nil {
		t.Fatal("expected readMapping error for incomplete mapping read")
	}
	if len(mr.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(mr.Events))
	}
	if !mr.Events[0].IsError() {
		t.Fatalf("expected read event to record error: %+v", mr.Events[0])
	}
	if len(mr.Blocks) != 1 {
		t.Fatalf("blocks = %d, want partial payload block", len(mr.Blocks))
	}
	if got, want := mr.Blocks[0].Data, []byte("abc"); !bytes.Equal(got, want) {
		t.Fatalf("partial block data = %q, want %q", got, want)
	}
	if mr.Blocks[0].Status != StatusError {
		t.Fatalf("partial block status = %d, want StatusError", mr.Blocks[0].Status)
	}
}

func TestAcquireNoMappings(t *testing.T) {
	skipIfNotLinux(t)
	pid := os.Getpid()

	// Filter that excludes everything.
	opts := Options{
		PID: pid,
		Filter: Filter{
			RequireReadable: true,
			PathnameMatch:   "this-does-not-exist",
		},
		ChunkSize: 4096,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := Acquire(ctx, opts)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success even with no mappings")
	}
	if len(result.Mappings) != 0 {
		t.Errorf("expected 0 mappings, got %d", len(result.Mappings))
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning for no mappings selected")
	}
}

type fakeProcessFS struct {
	maps    []procfs.Mapping
	mem     *scriptedMem
	openErr error
	mapsErr error
}

func (f fakeProcessFS) OpenMem(pid int) (processMemory, error) {
	if f.openErr != nil {
		return nil, f.openErr
	}
	return f.mem, nil
}

func (f fakeProcessFS) ReadMaps(pid int) ([]procfs.Mapping, error) {
	if f.mapsErr != nil {
		return nil, f.mapsErr
	}
	return f.maps, nil
}

type scriptedRead struct {
	data []byte
	err  error
}

type scriptedMem struct {
	reads       map[int64]scriptedRead
	readOffsets map[int64]bool
}

func (m *scriptedMem) ReadAt(p []byte, off int64) (int, error) {
	if m.readOffsets == nil {
		m.readOffsets = make(map[int64]bool)
	}
	m.readOffsets[off] = true
	read := m.reads[off]
	n := copy(p, read.data)
	return n, read.err
}

func (m *scriptedMem) Close() error {
	return nil
}
