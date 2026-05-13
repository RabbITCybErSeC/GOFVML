package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunConvertsRawToLiMEFile(t *testing.T) {
	tmp := t.TempDir()
	input := filepath.Join(tmp, "memory.raw")
	output := filepath.Join(tmp, "memory.lime")
	if err := os.WriteFile(input, []byte("GOFVML-RAW"), 0600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"-input", input,
		"-output", output,
		"-from", "raw",
		"-to", "lime",
		"-skip-zero=false",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Conversion complete: "+output) {
		t.Fatalf("stdout = %q, want conversion summary", stdout.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("GOFVML-RAW")) {
		t.Fatalf("converted output does not contain raw payload: %x", data)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("output mode = %v, want 0600", got)
	}
}

func TestRunRejectsMissingRequiredFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "-input, -output, and -to are required") {
		t.Fatalf("stderr = %q, want required flag message", stderr.String())
	}
}

func TestRunRejectsExistingOutput(t *testing.T) {
	tmp := t.TempDir()
	input := filepath.Join(tmp, "memory.raw")
	output := filepath.Join(tmp, "memory.lime")
	if err := os.WriteFile(input, []byte("payload"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("do not overwrite"), 0600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-input", input, "-output", output, "-from", "raw", "-to", "lime"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code for existing output")
	}
	if !strings.Contains(stderr.String(), "cannot create output") {
		t.Fatalf("stderr = %q, want create output error", stderr.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "do not overwrite" {
		t.Fatalf("existing output was overwritten: %q", data)
	}
}
