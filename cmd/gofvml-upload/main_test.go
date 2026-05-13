package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUploadsFileWithPUT(t *testing.T) {
	payload := []byte("GOFVML upload payload")
	var gotMethod string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	tmp := t.TempDir()
	file := filepath.Join(tmp, "memory.lime")
	if err := os.WriteFile(file, payload, 0600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-file", file, "-url", server.URL + "/upload", "-retries", "0"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotMethod != http.MethodPut {
		t.Fatalf("method = %s, want PUT", gotMethod)
	}
	if !bytes.Equal(gotBody, payload) {
		t.Fatalf("body = %q, want payload", gotBody)
	}
	if !strings.Contains(stdout.String(), "Upload complete: "+file) {
		t.Fatalf("stdout = %q, want upload summary", stdout.String())
	}
}

func TestRunDeleteAfterRemovesUploadedFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tmp := t.TempDir()
	file := filepath.Join(tmp, "memory.lime")
	if err := os.WriteFile(file, []byte("payload"), 0600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-file", file, "-url", server.URL, "-delete-after", "-retries", "0"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stderr = %s", code, stderr.String())
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("uploaded file still exists or stat failed with unexpected error: %v", err)
	}
}

func TestRunRejectsMissingRequiredFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "-file and -url are required") {
		t.Fatalf("stderr = %q, want required flag message", stderr.String())
	}
}
