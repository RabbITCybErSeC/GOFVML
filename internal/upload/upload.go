// Package upload provides upload workflow implementations.
package upload

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
	"github.com/RabbITCybErSeC/gofvml/internal/progress"
)

// Options configures an upload operation.
type Options struct {
	// FilePath is the local file to upload.
	FilePath string
	// URL is the destination URL for the HTTP PUT request.
	URL string
	// DeleteAfter, if true, deletes the local file after successful upload.
	DeleteAfter bool
	// Progress is an optional progress callback.
	Progress progress.Callback
	// Retries is the number of retry attempts for transient failures.
	Retries int
	// HTTPClient overrides the default HTTP client. If nil, a default client
	// with 30-second timeout is used.
	HTTPClient *http.Client
}

// Result holds the outcome of an upload operation.
type Result struct {
	// Success is true if upload completed successfully.
	Success bool
	// BytesUploaded is the number of bytes sent.
	BytesUploaded int64
	// URL is the destination URL.
	URL string
	// Warnings contains non-fatal issues.
	Warnings []*diagnostic.Diagnostic
}

// AddWarning appends a warning to the result.
func (r *Result) AddWarning(w *diagnostic.Diagnostic) {
	r.Warnings = append(r.Warnings, w)
}

// Upload uploads a local file to the given URL via HTTP PUT.
// It preserves the local file on failure or cancellation.
func Upload(ctx context.Context, opts Options) (*Result, error) {
	if opts.Retries <= 0 {
		opts.Retries = 3
	}

	result := &Result{URL: opts.URL}

	// Open the file.
	file, err := os.Open(opts.FilePath)
	if err != nil {
		return nil, diagnostic.UploadError("cannot open file").
			WithOperation("upload").
			WithTarget(opts.FilePath).
			WithCause(err)
	}
	defer file.Close()

	// Get file info for content length.
	stat, err := file.Stat()
	if err != nil {
		return nil, diagnostic.UploadError("cannot stat file").
			WithOperation("upload").
			WithTarget(opts.FilePath).
			WithCause(err)
	}
	fileSize := stat.Size()

	// Create progress-tracking reader.
	reporter := progress.NewReporter(opts.Progress)
	defer reporter.Close()

	var lastErr error
	for attempt := 0; attempt <= opts.Retries; attempt++ {
		if attempt > 0 {
			// Reset file position for retry.
			if _, err := file.Seek(0, io.SeekStart); err != nil {
				return nil, diagnostic.UploadError("cannot seek file for retry").
					WithOperation("upload").
					WithTarget(opts.FilePath).
					WithCause(err)
			}
			// Exponential backoff: 1s, 2s, 4s, ...
			delay := time.Duration(1<<uint(attempt-1)) * time.Second
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		pr := &progressReader{
			Reader:   file,
			Total:    fileSize,
			Reporter: reporter,
			URL:      opts.URL,
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, opts.URL, pr)
		if err != nil {
			return nil, diagnostic.UploadError("cannot create request").
				WithOperation("upload").
				WithTarget(opts.URL).
				WithCause(err)
		}

		req.ContentLength = fileSize
		req.Header.Set("Content-Type", "application/octet-stream")

		client := opts.HTTPClient
		if client == nil {
			client = &http.Client{Timeout: 30 * time.Second}
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			result.AddWarning(diagnostic.Warning(diagnostic.CategoryUpload, "upload attempt failed").
				WithOperation("upload").
				WithTarget(opts.URL).
				WithCause(err))
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			result.Success = true
			result.BytesUploaded = fileSize

			// Delete local file if requested.
			if opts.DeleteAfter {
				if err := os.Remove(opts.FilePath); err != nil {
					result.AddWarning(diagnostic.Warning(diagnostic.CategoryUpload, "failed to delete local file after upload").
						WithOperation("upload").
						WithTarget(opts.FilePath).
						WithCause(err))
				}
			}

			return result, nil
		}

		lastErr = fmt.Errorf("HTTP %d %s", resp.StatusCode, resp.Status)
		result.AddWarning(diagnostic.Warning(diagnostic.CategoryUpload, "upload rejected").
			WithOperation("upload").
			WithTarget(opts.URL).
			WithCause(lastErr))

		// Don't retry 4xx errors (client errors).
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			break
		}
	}

	return nil, diagnostic.UploadError("upload failed after all retries").
		WithOperation("upload").
		WithTarget(opts.URL).
		WithCause(lastErr)
}

type progressReader struct {
	Reader   io.Reader
	Total    int64
	Reporter *progress.Reporter
	URL      string
	read     int64
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.read += int64(n)
	pr.Reporter.Report(progress.Event{
		Operation: "upload",
		Phase:     "uploading",
		Current:   uint64(pr.read),
		Total:     uint64(pr.Total),
		Target:    pr.URL,
	})
	return n, err
}
