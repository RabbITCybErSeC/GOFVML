package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/RabbITCybErSeC/gofvml/internal/cli"
	"github.com/RabbITCybErSeC/gofvml/internal/progress"
	"github.com/RabbITCybErSeC/gofvml/internal/upload"
)

func main() {
	fs := flag.NewFlagSet("gofvml-upload", flag.ExitOnError)
	file := fs.String("file", "", "Local file path to upload (required)")
	url := fs.String("url", "", "Destination URL for HTTP PUT (required)")
	deleteAfter := fs.Bool("delete-after", false, "Delete local file after successful upload")
	progressFlag := fs.Bool("progress", false, "Show progress on stderr")
	retries := fs.Int("retries", 3, "Number of retry attempts")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gofvml-upload [options]\n\n")
		fmt.Fprintf(os.Stderr, "Upload memory images via HTTP PUT.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  gofvml-upload -file mem.lime -url https://example.com/upload/mem.lime\n")
		fmt.Fprintf(os.Stderr, "  gofvml-upload -file mem.lime -url https://example.com/upload/ -delete-after\n")
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *file == "" || *url == "" {
		fmt.Fprintln(os.Stderr, "Error: -file and -url are required")
		fs.Usage()
		os.Exit(1)
	}

	var progressCb progress.Callback
	if *progressFlag {
		progressCb = func(e progress.Event) {
			fmt.Fprintf(os.Stderr, "\r%s", e.String())
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
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		if result != nil {
			cli.RenderDiagnostics(os.Stderr, result.Warnings)
		}
		os.Exit(1)
	}

	if *progressFlag {
		fmt.Fprintln(os.Stderr)
	}

	fmt.Printf("Upload complete: %s\n", *file)
	fmt.Printf("  Destination: %s\n", result.URL)
	fmt.Printf("  Bytes uploaded: %d\n", result.BytesUploaded)

	cli.RenderDiagnostics(os.Stdout, result.Warnings)

	os.Exit(0)
}
