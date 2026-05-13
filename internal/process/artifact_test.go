package process

import (
	"bytes"
	"testing"
	"time"

	"github.com/RabbITCybErSeC/gofvml/internal/procfs"
)

func TestArtifactRoundTrip(t *testing.T) {
	blocks := []PayloadBlock{
		{
			VirtualAddress:  0x1000,
			MappingIndex:    0,
			CompressionType: CompressionNone,
			Status:          StatusOK,
			Data:            []byte("hello world"),
		},
		{
			VirtualAddress:  0x2000,
			MappingIndex:    1,
			CompressionType: CompressionNone,
			Status:          StatusShortRead,
			Data:            []byte("short"),
		},
	}

	meta := ArtifactMetadata{
		Timestamp:      time.Now().UTC().Truncate(time.Millisecond),
		PID:            1234,
		CommandLine:    []string{"/bin/sh", "-c", "echo hello"},
		ExecutablePath: "/bin/sh",
		Mappings: []procfs.Mapping{
			{Start: 0x1000, End: 0x2000, Perms: "r-xp", Pathname: "/bin/sh"},
			{Start: 0x2000, End: 0x3000, Perms: "rw-p", Pathname: "[heap]"},
		},
		ReadEvents: []ReadEvent{
			{VirtualAddress: 0x1000, Requested: 4096, Read: 4096},
			{VirtualAddress: 0x2000, Requested: 4096, Read: 5},
		},
		Strict:    false,
		BytesRead: 16,
	}

	var buf bytes.Buffer
	if err := WriteArtifact(&buf, blocks, meta); err != nil {
		t.Fatalf("WriteArtifact failed: %v", err)
	}

	// Read back.
	readMeta, readBlocks, err := ReadArtifact(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("ReadArtifact failed: %v", err)
	}

	if len(readBlocks) != len(blocks) {
		t.Fatalf("expected %d blocks, got %d", len(blocks), len(readBlocks))
	}
	for i, b := range readBlocks {
		if b.VirtualAddress != blocks[i].VirtualAddress {
			t.Errorf("block[%d].VirtualAddress = %x, want %x", i, b.VirtualAddress, blocks[i].VirtualAddress)
		}
		if b.MappingIndex != blocks[i].MappingIndex {
			t.Errorf("block[%d].MappingIndex = %d, want %d", i, b.MappingIndex, blocks[i].MappingIndex)
		}
		if b.CompressionType != blocks[i].CompressionType {
			t.Errorf("block[%d].CompressionType = %d, want %d", i, b.CompressionType, blocks[i].CompressionType)
		}
		if b.Status != blocks[i].Status {
			t.Errorf("block[%d].Status = %d, want %d", i, b.Status, blocks[i].Status)
		}
		if !bytes.Equal(b.Data, blocks[i].Data) {
			t.Errorf("block[%d].Data = %q, want %q", i, b.Data, blocks[i].Data)
		}
	}

	if readMeta.PID != meta.PID {
		t.Errorf("meta.PID = %d, want %d", readMeta.PID, meta.PID)
	}
	if !readMeta.Timestamp.Equal(meta.Timestamp) {
		t.Errorf("meta.Timestamp = %v, want %v", readMeta.Timestamp, meta.Timestamp)
	}
	if len(readMeta.CommandLine) != len(meta.CommandLine) {
		t.Errorf("meta.CommandLine len = %d, want %d", len(readMeta.CommandLine), len(meta.CommandLine))
	}
	if readMeta.ExecutablePath != meta.ExecutablePath {
		t.Errorf("meta.ExecutablePath = %q, want %q", readMeta.ExecutablePath, meta.ExecutablePath)
	}
	if len(readMeta.Mappings) != len(meta.Mappings) {
		t.Errorf("meta.Mappings len = %d, want %d", len(readMeta.Mappings), len(meta.Mappings))
	}
	if readMeta.BytesRead != meta.BytesRead {
		t.Errorf("meta.BytesRead = %d, want %d", readMeta.BytesRead, meta.BytesRead)
	}
	if readMeta.Version != "gofvml-process-v1" {
		t.Errorf("meta.Version = %q, want %q", readMeta.Version, "gofvml-process-v1")
	}
}

func TestArtifactEmptyBlocks(t *testing.T) {
	meta := ArtifactMetadata{
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
		PID:       5678,
		Mappings:  nil,
		BytesRead: 0,
	}

	var buf bytes.Buffer
	if err := WriteArtifact(&buf, nil, meta); err != nil {
		t.Fatalf("WriteArtifact failed: %v", err)
	}

	readMeta, readBlocks, err := ReadArtifact(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("ReadArtifact failed: %v", err)
	}

	if len(readBlocks) != 0 {
		t.Errorf("expected 0 blocks, got %d", len(readBlocks))
	}
	if readMeta.PID != 5678 {
		t.Errorf("meta.PID = %d, want 5678", readMeta.PID)
	}
}

func TestArtifactInvalidMagic(t *testing.T) {
	data := make([]byte, ArtifactHeaderSize)
	copy(data, []byte("INVALID!!"))

	_, err := DecodeArtifactHeader(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for invalid magic")
	}
}

func TestArtifactInvalidVersion(t *testing.T) {
	data := make([]byte, ArtifactHeaderSize)
	copy(data, []byte(ArtifactMagic))
	data[9] = 99 // invalid version

	_, err := DecodeArtifactHeader(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for invalid version")
	}
}

func TestArtifactTruncatedFooter(t *testing.T) {
	var buf bytes.Buffer
	_ = EncodeArtifactHeader(&buf)
	// Write incomplete footer
	buf.Write([]byte("incomplete"))

	_, _, err := ReadArtifact(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("expected error for truncated footer")
	}
}

func TestArtifactLargeBlock(t *testing.T) {
	// Test with a block larger than typical chunk size.
	largeData := make([]byte, 1024*1024) // 1 MiB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	blocks := []PayloadBlock{
		{
			VirtualAddress:  0x1000,
			MappingIndex:    0,
			CompressionType: CompressionNone,
			Status:          StatusOK,
			Data:            largeData,
		},
	}

	meta := ArtifactMetadata{
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
		PID:       9999,
		BytesRead: uint64(len(largeData)),
	}

	var buf bytes.Buffer
	if err := WriteArtifact(&buf, blocks, meta); err != nil {
		t.Fatalf("WriteArtifact failed: %v", err)
	}

	readMeta, readBlocks, err := ReadArtifact(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("ReadArtifact failed: %v", err)
	}

	if len(readBlocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(readBlocks))
	}
	if !bytes.Equal(readBlocks[0].Data, largeData) {
		t.Error("large block data mismatch")
	}
	if readMeta.BytesRead != uint64(len(largeData)) {
		t.Errorf("meta.BytesRead = %d, want %d", readMeta.BytesRead, len(largeData))
	}
}

func TestArtifactMappingRoundTrip(t *testing.T) {
	// Verify that mapping metadata survives round trip.
	mappings := []procfs.Mapping{
		{
			Start:    0x55f3c1a2b000,
			End:      0x55f3c1a2c000,
			Perms:    "r--p",
			Offset:   0,
			DevMajor: 8,
			DevMinor: 1,
			Inode:    1310734,
			Pathname: "/usr/bin/cat",
		},
		{
			Start:    0x7ffd9c6a3000,
			End:      0x7ffd9c6c4000,
			Perms:    "rw-p",
			Offset:   0,
			DevMajor: 0,
			DevMinor: 0,
			Inode:    0,
			Pathname: "[stack]",
		},
	}

	meta := ArtifactMetadata{
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
		PID:       42,
		Mappings:  mappings,
		BytesRead: 0,
	}

	var buf bytes.Buffer
	if err := WriteArtifact(&buf, nil, meta); err != nil {
		t.Fatalf("WriteArtifact failed: %v", err)
	}

	readMeta, _, err := ReadArtifact(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("ReadArtifact failed: %v", err)
	}

	if len(readMeta.Mappings) != len(mappings) {
		t.Fatalf("expected %d mappings, got %d", len(mappings), len(readMeta.Mappings))
	}
	for i, m := range readMeta.Mappings {
		if m.Start != mappings[i].Start {
			t.Errorf("mapping[%d].Start = %x, want %x", i, m.Start, mappings[i].Start)
		}
		if m.End != mappings[i].End {
			t.Errorf("mapping[%d].End = %x, want %x", i, m.End, mappings[i].End)
		}
		if m.Perms != mappings[i].Perms {
			t.Errorf("mapping[%d].Perms = %q, want %q", i, m.Perms, mappings[i].Perms)
		}
		if m.Offset != mappings[i].Offset {
			t.Errorf("mapping[%d].Offset = %x, want %x", i, m.Offset, mappings[i].Offset)
		}
		if m.DevMajor != mappings[i].DevMajor {
			t.Errorf("mapping[%d].DevMajor = %d, want %d", i, m.DevMajor, mappings[i].DevMajor)
		}
		if m.DevMinor != mappings[i].DevMinor {
			t.Errorf("mapping[%d].DevMinor = %d, want %d", i, m.DevMinor, mappings[i].DevMinor)
		}
		if m.Inode != mappings[i].Inode {
			t.Errorf("mapping[%d].Inode = %d, want %d", i, m.Inode, mappings[i].Inode)
		}
		if m.Pathname != mappings[i].Pathname {
			t.Errorf("mapping[%d].Pathname = %q, want %q", i, m.Pathname, mappings[i].Pathname)
		}
	}
}
