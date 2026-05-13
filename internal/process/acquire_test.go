package process

import (
	"bytes"
	"context"
	"os"
	"runtime"
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
