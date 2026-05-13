package sidecar

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
	"github.com/RabbITCybErSeC/gofvml/internal/phys"
)

func TestWritePhysicalSidecar(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	meta := PhysicalMetadata{
		Host:          "testhost",
		Kernel:        "5.15.0-generic",
		Source:        "kcore",
		Format:        "lime",
		Ranges:        []phys.Range{{Start: 0x1000, End: 0x2000}},
		TotalBytes:    0x1000,
		Warnings:      []*diagnostic.Diagnostic{diagnostic.Warning(diagnostic.CategorySource, "test warning")},
		VolatilityHints: VolatilityHints{Profile: "LinuxUbuntu_5_15_0"},
	}

	if err := WritePhysicalSidecar(path, meta); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var readMeta PhysicalMetadata
	if err := json.Unmarshal(data, &readMeta); err != nil {
		t.Fatalf("unmarshal sidecar: %v", err)
	}

	if readMeta.Version != "gofvml-sidecar-v1" {
		t.Errorf("expected version gofvml-sidecar-v1, got %s", readMeta.Version)
	}
	if readMeta.Host != "testhost" {
		t.Errorf("expected host testhost, got %s", readMeta.Host)
	}
	if readMeta.Source != "kcore" {
		t.Errorf("expected source kcore, got %s", readMeta.Source)
	}
	if len(readMeta.Ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(readMeta.Ranges))
	}
	if readMeta.Ranges[0].Start != 0x1000 || readMeta.Ranges[0].End != 0x2000 {
		t.Errorf("unexpected range: %+v", readMeta.Ranges[0])
	}
	if len(readMeta.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(readMeta.Warnings))
	}
	if readMeta.VolatilityHints.Profile != "LinuxUbuntu_5_15_0" {
		t.Errorf("expected profile LinuxUbuntu_5_15_0, got %s", readMeta.VolatilityHints.Profile)
	}
}

func TestWritePhysicalSidecarToWriter(t *testing.T) {
	meta := PhysicalMetadata{
		Host:   "testhost",
		Source: "crash",
		Format: "avml",
	}

	var buf bytes.Buffer
	if err := WritePhysicalSidecarToWriter(&buf, meta); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var readMeta PhysicalMetadata
	if err := json.Unmarshal(buf.Bytes(), &readMeta); err != nil {
		t.Fatalf("unmarshal sidecar: %v", err)
	}

	if readMeta.Host != "testhost" {
		t.Errorf("expected host testhost, got %s", readMeta.Host)
	}
}

func TestWritePhysicalSidecar_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	meta := PhysicalMetadata{}
	if err := WritePhysicalSidecar(path, meta); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var readMeta PhysicalMetadata
	if err := json.Unmarshal(data, &readMeta); err != nil {
		t.Fatalf("unmarshal sidecar: %v", err)
	}

	if readMeta.Version != "gofvml-sidecar-v1" {
		t.Errorf("expected version gofvml-sidecar-v1, got %s", readMeta.Version)
	}
	if readMeta.GOFVMLVersion != "0.1.0" {
		t.Errorf("expected GOFVMLVersion 0.1.0, got %s", readMeta.GOFVMLVersion)
	}
	if readMeta.VolatilityHints.OS != "linux" {
		t.Errorf("expected OS linux, got %s", readMeta.VolatilityHints.OS)
	}
	if readMeta.GeneratedAt.IsZero() {
		t.Error("expected GeneratedAt to be set")
	}
}

func TestWritePhysicalSidecar_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	// Create existing file.
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}

	meta := PhysicalMetadata{}
	err := WritePhysicalSidecar(path, meta)
	if err == nil {
		t.Fatal("expected error for existing file")
	}
}

func TestPhysicalMetadata_JSONRoundTrip(t *testing.T) {
	meta := PhysicalMetadata{
		Version:       "gofvml-sidecar-v1",
		GeneratedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Host:          "testhost",
		Kernel:        "6.1.0",
		GOFVMLVersion: "0.1.0",
		Source:        "mem",
		Format:        "raw",
		Ranges: []phys.Range{
			{Start: 0, End: 0x1000},
			{Start: 0x10000, End: 0x20000},
		},
		TotalBytes: 0x11000,
		Timing: TimingInfo{
			StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
			Duration:  time.Minute,
		},
		Warnings: []*diagnostic.Diagnostic{
			diagnostic.Warning(diagnostic.CategorySource, "warning 1"),
			diagnostic.Warning(diagnostic.CategoryPolicy, "warning 2"),
		},
		VolatilityHints: VolatilityHints{
			Profile:      "LinuxDebian_6_1_0",
			OS:           "linux",
			Architecture: "amd64",
			Notes:        "Test notes",
		},
	}

	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded PhysicalMetadata
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Host != meta.Host {
		t.Errorf("host mismatch: %s vs %s", decoded.Host, meta.Host)
	}
	if decoded.TotalBytes != meta.TotalBytes {
		t.Errorf("total_bytes mismatch: %d vs %d", decoded.TotalBytes, meta.TotalBytes)
	}
	if len(decoded.Ranges) != len(meta.Ranges) {
		t.Errorf("ranges length mismatch: %d vs %d", len(decoded.Ranges), len(meta.Ranges))
	}
	if len(decoded.Warnings) != len(meta.Warnings) {
		t.Errorf("warnings length mismatch: %d vs %d", len(decoded.Warnings), len(meta.Warnings))
	}
}
