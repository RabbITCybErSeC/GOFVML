package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/RabbITCybErSeC/gofvml/internal/cli"
	"github.com/RabbITCybErSeC/gofvml/internal/progress"
	"github.com/RabbITCybErSeC/gofvml/internal/upload"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gofvml-upload", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file := fs.String("file", "", "Local file path to upload (required)")
	url := fs.String("url", "", "Destination URL for HTTP PUT (required)")
	deleteAfter := fs.Bool("delete-after", false, "Delete local file after successful upload")
	progressFlag := fs.Bool("progress", false, "Show progress on stderr")
	retries := fs.Int("retries", 3, "Number of retry attempts")

	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: gofvml-upload [options]\n\n")
		fmt.Fprintf(stderr, "Upload memory images via HTTP PUT.\n\n")
		fmt.Fprintf(stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(stderr, "\nExamples:\n")
		fmt.Fprintf(stderr, "  gofvml-upload -file mem.lime -url https://example.com/upload/mem.lime\n")
		fmt.Fprintf(stderr, "  gofvml-upload -file mem.lime -url https://example.com/upload/ -delete-after\n")
	}

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	if *file == "" || *url == "" {
		fmt.Fprintln(stderr, "Error: -file and -url are required")
		fs.Usage()
		return 1
	}

	var progressCb progress.Callback
	if *progressFlag {
		progressCb = func(e progress.Event) {
			fmt.Fprintf(stderr, "\r%s", e.String())
		}
	}

	opts := upload.Options{
		FilePath:    *file,
		URL:         *url,
		DeleteAfter: *deleteAfter,
		Progress:    progressCb,
		Retries:     *retries,
	}

	ctx := context.Background()
	result, err := upload.Upload(ctx, opts)
	if err != nil {
		fmt.Fprintf(stderr, "\nError: %v\n", err)
		if result != nil {
			cli.RenderDiagnostics(stderr, result.Warnings)
		}
		return 1
	}

	if *progressFlag {
		fmt.Fprintln(stderr)
	}

	fmt.Fprintf(stdout, "Upload complete: %s\n", *file)
	fmt.Fprintf(stdout, "  Destination: %s\n", result.URL)
	fmt.Fprintf(stdout, "  Bytes uploaded: %d\n", result.BytesUploaded)

	cli.RenderDiagnostics(stdout, result.Warnings)

	return 0
}
