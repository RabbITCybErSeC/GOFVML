// Package sidecar provides metadata sidecar generation for memory acquisitions.
package sidecar

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
	"github.com/RabbITCybErSeC/gofvml/internal/phys"
)

// PhysicalMetadata is the sidecar schema for physical memory acquisitions.
type PhysicalMetadata struct {
	// Version is the sidecar schema version.
	Version string `json:"version"`
	// GeneratedAt is the timestamp when the sidecar was created.
	GeneratedAt time.Time `json:"generated_at"`
	// Host is the hostname where acquisition ran.
	Host string `json:"host,omitempty"`
	// Kernel is the Linux kernel version.
	Kernel string `json:"kernel,omitempty"`
	// GOFVMLVersion is the GOFVML tool version.
	GOFVMLVersion string `json:"gofvml_version,omitempty"`
	// Source is the memory source used.
	Source string `json:"source,omitempty"`
	// Format is the output image format.
	Format string `json:"format,omitempty"`
	// Ranges are the physical memory ranges acquired.
	Ranges []phys.Range `json:"ranges,omitempty"`
	// TotalBytes is the sum of range lengths.
	TotalBytes uint64 `json:"total_bytes"`
	// Timing holds operation timing.
	Timing TimingInfo `json:"timing,omitempty"`
	// Warnings holds non-fatal issues.
	Warnings []*diagnostic.Diagnostic `json:"warnings,omitempty"`
	// VolatilityHints provides Volatility-oriented guidance.
	VolatilityHints VolatilityHints `json:"volatility_hints,omitempty"`
}

// TimingInfo holds timing information for the acquisition.
type TimingInfo struct {
	// StartTime is when acquisition began.
	StartTime time.Time `json:"start_time,omitempty"`
	// EndTime is when acquisition completed.
	EndTime time.Time `json:"end_time,omitempty"`
	// Duration is the total acquisition time.
	Duration time.Duration `json:"duration,omitempty"`
}

// VolatilityHints provides guidance for Volatility 3 analysis.
type VolatilityHints struct {
	// Profile is the suggested Volatility profile or symbol table.
	Profile string `json:"profile,omitempty"`
	// OS is the operating system identifier.
	OS string `json:"os,omitempty"`
	// Architecture is the CPU architecture.
	Architecture string `json:"architecture,omitempty"`
	// Notes contains additional analyst guidance.
	Notes string `json:"notes,omitempty"`
}

// WritePhysicalSidecar writes a physical metadata sidecar to the given path.
// The sidecar path is typically the image path with ".json" appended.
func WritePhysicalSidecar(path string, meta PhysicalMetadata) error {
	meta.Version = "gofvml-sidecar-v1"
	if meta.GeneratedAt.IsZero() {
		meta.GeneratedAt = time.Now()
	}
	if meta.GOFVMLVersion == "" {
		meta.GOFVMLVersion = "0.1.0"
	}
	if meta.VolatilityHints.OS == "" {
		meta.VolatilityHints.OS = "linux"
	}
	if meta.VolatilityHints.Architecture == "" {
		meta.VolatilityHints.Architecture = runtime.GOARCH
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sidecar: %w", err)
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("create sidecar: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write sidecar: %w", err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("write sidecar newline: %w", err)
	}

	return nil
}

// WritePhysicalSidecarToWriter writes a physical metadata sidecar to an io.Writer.
func WritePhysicalSidecarToWriter(w io.Writer, meta PhysicalMetadata) error {
	meta.Version = "gofvml-sidecar-v1"
	if meta.GeneratedAt.IsZero() {
		meta.GeneratedAt = time.Now()
	}
	if meta.GOFVMLVersion == "" {
		meta.GOFVMLVersion = "0.1.0"
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sidecar: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write sidecar: %w", err)
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return fmt.Errorf("write sidecar newline: %w", err)
	}

	return nil
}
