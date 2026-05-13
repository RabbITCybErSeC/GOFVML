package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/RabbITCybErSeC/gofvml/internal/cli"
	"github.com/RabbITCybErSeC/gofvml/internal/conversion"
	"github.com/RabbITCybErSeC/gofvml/internal/progress"
)

func main() {
	fs := flag.NewFlagSet("gofvml-convert", flag.ExitOnError)
	input := fs.String("input", "", "Input file path (required)")
	output := fs.String("output", "", "Output file path (required)")
	fromFormat := fs.String("from", "", "Source format: raw, lime, or avml (auto-detect if empty)")
	toFormat := fs.String("to", "", "Target format: raw, lime, or avml (required)")
	skipZero := fs.Bool("skip-zero", true, "Skip all-zero chunks")
	progressFlag := fs.Bool("progress", false, "Show progress on stderr")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gofvml-convert [options]\n\n")
		fmt.Fprintf(os.Stderr, "Convert between memory image formats.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  gofvml-convert -input mem.raw -output mem.lime -to lime\n")
		fmt.Fprintf(os.Stderr, "  gofvml-convert -input mem.lime -output mem.avml -from lime -to avml\n")
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *input == "" || *output == "" || *toFormat == "" {
		fmt.Fprintln(os.Stderr, "Error: -input, -output, and -to are required")
		fs.Usage()
		os.Exit(1)
	}

	// Parse formats.
	var sourceFormat conversion.Format
	switch strings.ToLower(*fromFormat) {
	case "":
		sourceFormat = conversion.FormatUnknown
	case "raw":
		sourceFormat = conversion.FormatRaw
	case "lime":
		sourceFormat = conversion.FormatLiME
	case "avml":
		sourceFormat = conversion.FormatAVML
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown source format %q\n", *fromFormat)
		os.Exit(1)
	}

	var targetFormat conversion.Format
	switch strings.ToLower(*toFormat) {
	case "raw":
		targetFormat = conversion.FormatRaw
	case "lime":
		targetFormat = conversion.FormatLiME
	case "avml":
		targetFormat = conversion.FormatAVML
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown target format %q\n", *toFormat)
		os.Exit(1)
	}

	// Open input.
	inFile, err := os.Open(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot open input: %v\n", err)
		os.Exit(1)
	}
	defer inFile.Close()

	// Create output safely.
	outFile, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot create output: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	var progressCb progress.Callback
	if *progressFlag {
		progressCb = func(e progress.Event) {
			fmt.Fprintf(os.Stderr, "\r%s", e.String())
		}
	}

	opts := conversion.Options{
		SourceFormat:   sourceFormat,
		TargetFormat:   targetFormat,
		SkipZeroChunks: *skipZero,
		Progress:       progressCb,
	}

	ctx := context.Background()
	result, err := conversion.Convert(ctx, inFile, outFile, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}

	if *progressFlag {
		fmt.Fprintln(os.Stderr)
	}

	fmt.Printf("Conversion complete: %s\n", *output)
	fmt.Printf("  Source format: %s\n", result.SourceFormat)
	fmt.Printf("  Target format: %s\n", result.TargetFormat)
	fmt.Printf("  Bytes read: %d\n", result.BytesRead)
	fmt.Printf("  Bytes written: %d\n", result.BytesWritten)
	fmt.Printf("  Chunks processed: %d\n", result.ChunksRead)
	if result.ChunksSkipped > 0 {
		fmt.Printf("  Chunks skipped: %d\n", result.ChunksSkipped)
	}

	cli.RenderDiagnostics(os.Stdout, result.Warnings)

	os.Exit(0)
}
