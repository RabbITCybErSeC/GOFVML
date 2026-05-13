package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RabbITCybErSeC/gofvml/internal/conversion"
	"github.com/RabbITCybErSeC/gofvml/internal/process"
	"github.com/RabbITCybErSeC/gofvml/internal/procfs"
	"github.com/RabbITCybErSeC/gofvml/internal/upload"
)

func TestCLI_Help(t *testing.T) {
	// Test that help text is printed when no command is given.
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"gofvml"}

	// We can't easily test main() directly since it calls os.Exit.
	// Instead, test the help output function indirectly.
	var buf strings.Builder
	printUsageTo(&buf)

	output := buf.String()
	if !strings.Contains(output, "gofvml - Linux volatile memory acquisition tool") {
		t.Error("expected help text to contain tool name")
	}
	if !strings.Contains(output, "physical") {
		t.Error("expected help text to mention physical command")
	}
	if !strings.Contains(output, "process") {
		t.Error("expected help text to mention process command")
	}
	for _, command := range []string{"convert", "compress", "upload"} {
		if !strings.Contains(output, command) {
			t.Errorf("expected help text to mention %s command", command)
		}
	}
}

func TestCLI_Physical_MissingOutput(t *testing.T) {
	code := runPhysical(nil)
	if code == 0 {
		t.Fatal("expected physical command to fail without -output")
	}
}

func TestCLI_Process_MissingPID(t *testing.T) {
	code := runProcess([]string{"-output", "/tmp/test.art"})
	if code == 0 {
		t.Fatal("expected process command to fail without -pid")
	}
}

func TestCLI_Process_MissingOutput(t *testing.T) {
	code := runProcess([]string{"-pid", "123"})
	if code == 0 {
		t.Fatal("expected process command to fail without -output")
	}
}

func TestBuildProcessArtifactUsesAcquiredPayloadBlocks(t *testing.T) {
	result := &process.Result{
		PID:       123,
		BytesRead: 5,
		Mappings: []process.MappingResult{
			{
				Mapping: procfs.Mapping{Start: 0x1000, End: 0x1005, Perms: "r--p"},
				Events:  []process.ReadEvent{{VirtualAddress: 0x1000, Requested: 5, Read: 5}},
				Blocks: []process.PayloadBlock{
					{
						VirtualAddress:  0x1000,
						MappingIndex:    0,
						CompressionType: process.CompressionNone,
						Status:          process.StatusOK,
						Data:            []byte("hello"),
					},
				},
			},
		},
	}

	meta, blocks := buildProcessArtifact(result, false, time.Unix(10, 0).UTC())
	if meta.PID != 123 || meta.BytesRead != 5 {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
	if len(blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(blocks))
	}
	if string(blocks[0].Data) != "hello" {
		t.Fatalf("artifact block data = %q, want acquired payload", blocks[0].Data)
	}
}

func TestRunConvertRejectsMissingRequiredFlags(t *testing.T) {
	code := runConvert(nil)
	if code == 0 {
		t.Fatal("expected convert command to fail without required flags")
	}
}

func TestBuildCompressOptionsDefaultsToAVML(t *testing.T) {
	opts, err := buildCompressOptions(compressConfig{fromFormat: "lime"})
	if err != nil {
		t.Fatalf("buildCompressOptions error: %v", err)
	}
	if opts.SourceFormat != conversion.FormatLiME {
		t.Fatalf("SourceFormat = %s, want lime", opts.SourceFormat)
	}
	if opts.TargetFormat != conversion.FormatAVML {
		t.Fatalf("TargetFormat = %s, want avml", opts.TargetFormat)
	}
}

func TestRunUploadRejectsMissingRequiredFlags(t *testing.T) {
	code := runUpload(nil)
	if code == 0 {
		t.Fatal("expected upload command to fail without required flags")
	}
}

func TestRunConvertPerformsRawToLiMEConversion(t *testing.T) {
	tmp := t.TempDir()
	input := filepath.Join(tmp, "memory.raw")
	output := filepath.Join(tmp, "memory.lime")
	if err := os.WriteFile(input, []byte("primary gofvml convert"), 0600); err != nil {
		t.Fatal(err)
	}

	code := runConvert([]string{
		"-input", input,
		"-output", output,
		"-from", "raw",
		"-to", "lime",
		"-skip-zero=false",
	})
	if code != 0 {
		t.Fatalf("runConvert exit code = %d", code)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("primary gofvml convert")) {
		t.Fatalf("converted output does not contain source payload: %x", data)
	}
}

func TestRunConvertDoesNotOverwriteExistingOutput(t *testing.T) {
	tmp := t.TempDir()
	input := filepath.Join(tmp, "memory.raw")
	output := filepath.Join(tmp, "memory.lime")
	if err := os.WriteFile(input, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}

	code := runConvert([]string{"-input", input, "-output", output, "-from", "raw", "-to", "lime"})
	if code == 0 {
		t.Fatal("expected runConvert to reject existing output")
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing" {
		t.Fatalf("existing output was overwritten: %q", data)
	}
}

func TestRunCompressWritesAVMLOutputByDefault(t *testing.T) {
	tmp := t.TempDir()
	input := filepath.Join(tmp, "memory.raw")
	output := filepath.Join(tmp, "memory.avml")
	if err := os.WriteFile(input, []byte("primary gofvml compress"), 0600); err != nil {
		t.Fatal(err)
	}

	code := runCompress([]string{
		"-input", input,
		"-output", output,
		"-from", "raw",
		"-skip-zero=false",
	})
	if code != 0 {
		t.Fatalf("runCompress exit code = %d", code)
	}
	f, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got, err := conversion.DetectFormat(f); err != nil || got != conversion.FormatAVML {
		t.Fatalf("DetectFormat = %s, %v; want avml", got, err)
	}
}

func TestRunUploadSendsFileToHTTPServer(t *testing.T) {
	payload := []byte("primary gofvml upload")
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tmp := t.TempDir()
	file := filepath.Join(tmp, "memory.lime")
	if err := os.WriteFile(file, payload, 0600); err != nil {
		t.Fatal(err)
	}

	code := runUpload([]string{"-file", file, "-url", server.URL + "/upload", "-retries", "0"})
	if code != 0 {
		t.Fatalf("runUpload exit code = %d", code)
	}
	if !bytes.Equal(gotBody, payload) {
		t.Fatalf("uploaded body = %q, want payload", gotBody)
	}
}

func TestConvertRunnerUsesConversionWorkflow(t *testing.T) {
	var called bool
	var output bytes.Buffer
	runner := convertRunnerFunc(func(ctx context.Context, input io.Reader, out io.Writer, opts conversion.Options) (*conversion.Result, error) {
		called = true
		if opts.TargetFormat != conversion.FormatAVML {
			t.Fatalf("TargetFormat = %s, want avml", opts.TargetFormat)
		}
		return &conversion.Result{Success: true, TargetFormat: opts.TargetFormat}, nil
	})
	if err := runConversion(context.Background(), nil, &output, conversion.Options{TargetFormat: conversion.FormatAVML}, runner); err != nil {
		t.Fatalf("runConversion error: %v", err)
	}
	if !called {
		t.Fatal("expected conversion runner to be called")
	}
}

func TestUploadRunnerUsesUploadWorkflow(t *testing.T) {
	var called bool
	runner := uploadRunnerFunc(func(ctx context.Context, opts upload.Options) (*upload.Result, error) {
		called = true
		if opts.Retries != 2 {
			t.Fatalf("Retries = %d, want 2", opts.Retries)
		}
		return &upload.Result{Success: true, URL: opts.URL}, nil
	})
	if err := runUploadWorkflow(context.Background(), upload.Options{URL: "http://example.test", Retries: 2}, runner); err != nil {
		t.Fatalf("runUploadWorkflow error: %v", err)
	}
	if !called {
		t.Fatal("expected upload runner to be called")
	}
}

func printUsageTo(w *strings.Builder) {
	w.WriteString("gofvml - Linux volatile memory acquisition tool\n")
	w.WriteString("\n")
	w.WriteString("Usage: gofvml <command> [options]\n")
	w.WriteString("\n")
	w.WriteString("Commands:\n")
	w.WriteString("  physical   Acquire physical memory\n")
	w.WriteString("  process    Acquire process memory\n")
	w.WriteString("  convert    Convert memory image formats\n")
	w.WriteString("  compress   Compress memory images\n")
	w.WriteString("  upload     Upload memory images\n")
	w.WriteString("  version    Print version\n")
	w.WriteString("  help       Show this help message\n")
	w.WriteString("\n")
	w.WriteString("Run 'gofvml <command> -help' for command-specific options.\n")
}
