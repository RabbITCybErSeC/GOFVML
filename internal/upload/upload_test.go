package upload

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RabbITCybErSeC/gofvml/internal/progress"
)

func TestUpload_Success(t *testing.T) {
	content := []byte("test upload content")
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != string(content) {
			t.Errorf("expected %q, got %q", content, body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, content, 0600); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		FilePath: filePath,
		URL:      server.URL + "/upload",
		Retries:  0,
	}

	ctx := context.Background()
	result, err := Upload(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success")
	}
	if result.BytesUploaded != int64(len(content)) {
		t.Errorf("expected %d bytes, got %d", len(content), result.BytesUploaded)
	}
}

func TestUpload_Rejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		FilePath: filePath,
		URL:      server.URL + "/upload",
		Retries:  0,
	}

	ctx := context.Background()
	_, err := Upload(ctx, opts)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpload_DeleteAfterSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		FilePath:    filePath,
		URL:         server.URL + "/upload",
		DeleteAfter: true,
		Retries:     0,
	}

	ctx := context.Background()
	_, err := Upload(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

func TestUpload_PreserveAfterFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		FilePath:    filePath,
		URL:         server.URL + "/upload",
		DeleteAfter: true,
		Retries:     0,
	}

	ctx := context.Background()
	_, err := Upload(ctx, opts)
	if err == nil {
		t.Fatal("expected error")
	}

	if _, err := os.Stat(filePath); err != nil {
		t.Errorf("expected file to be preserved: %v", err)
	}
}

func TestUpload_Progress(t *testing.T) {
	content := make([]byte, 1024)
	for i := range content {
		content[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.bin")
	if err := os.WriteFile(filePath, content, 0600); err != nil {
		t.Fatal(err)
	}

	var events []progress.Event
	progressCb := func(e progress.Event) {
		events = append(events, e)
	}

	opts := Options{
		FilePath: filePath,
		URL:      server.URL + "/upload",
		Progress: progressCb,
		Retries:  0,
	}

	ctx := context.Background()
	_, err := Upload(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) == 0 {
		t.Error("expected progress events")
	}

	// Check that we got events showing progress.
	var lastEvent progress.Event
	if len(events) > 0 {
		lastEvent = events[len(events)-1]
	}
	if lastEvent.Current != uint64(len(content)) {
		t.Errorf("expected final progress %d, got %d", len(content), lastEvent.Current)
	}
}

func TestUpload_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		FilePath: filePath,
		URL:      server.URL + "/upload",
		Retries:  3,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx := context.Background()
	result, err := Upload(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success after retry")
	}
	if attempts < 2 {
		t.Errorf("expected at least 2 attempts, got %d", attempts)
	}
}

func TestUpload_Cancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response to allow cancellation.
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	opts := Options{
		FilePath: filePath,
		URL:      server.URL + "/upload",
		Retries:  0,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}

	_, err := Upload(ctx, opts)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestUpload_MissingFile(t *testing.T) {
	opts := Options{
		FilePath: "/nonexistent/file.txt",
		URL:      "http://example.com/upload",
		Retries:  0,
	}

	ctx := context.Background()
	_, err := Upload(ctx, opts)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestUpload_NoRetryOn4xx(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		FilePath: filePath,
		URL:      server.URL + "/upload",
		Retries:  3,
	}

	ctx := context.Background()
	_, err := Upload(ctx, opts)
	if err == nil {
		t.Fatal("expected error")
	}

	if attempts != 1 {
		t.Errorf("expected 1 attempt for 4xx, got %d", attempts)
	}
}
