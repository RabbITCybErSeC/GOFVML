package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/RabbITCybErSeC/gofvml/internal/cli"
	"github.com/RabbITCybErSeC/gofvml/internal/conversion"
	"github.com/RabbITCybErSeC/gofvml/internal/progress"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gofvml-convert", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := fs.String("input", "", "Input file path (required)")
	output := fs.String("output", "", "Output file path (required)")
	fromFormat := fs.String("from", "", "Source format: raw, lime, or avml (auto-detect if empty)")
	toFormat := fs.String("to", "", "Target format: raw, lime, or avml (required)")
	skipZero := fs.Bool("skip-zero", true, "Skip all-zero chunks")
	progressFlag := fs.Bool("progress", false, "Show progress on stderr")

	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: gofvml-convert [options]\n\n")
		fmt.Fprintf(stderr, "Convert between memory image formats.\n\n")
		fmt.Fprintf(stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(stderr, "\nExamples:\n")
		fmt.Fprintf(stderr, "  gofvml-convert -input mem.raw -output mem.lime -to lime\n")
		fmt.Fprintf(stderr, "  gofvml-convert -input mem.lime -output mem.avml -from lime -to avml\n")
	}

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	if *input == "" || *output == "" || *toFormat == "" {
		fmt.Fprintln(stderr, "Error: -input, -output, and -to are required")
		fs.Usage()
		return 1
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
		fmt.Fprintf(stderr, "Error: unknown source format %q\n", *fromFormat)
		return 1
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
		fmt.Fprintf(stderr, "Error: unknown target format %q\n", *toFormat)
		return 1
	}

	// Open input.
	inFile, err := os.Open(*input)
	if err != nil {
		fmt.Fprintf(stderr, "Error: cannot open input: %v\n", err)
		return 1
	}
	defer inFile.Close()

	// Create output safely.
	outFile, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Fprintf(stderr, "Error: cannot create output: %v\n", err)
		return 1
	}
	defer outFile.Close()

	var progressCb progress.Callback
	if *progressFlag {
		progressCb = func(e progress.Event) {
			fmt.Fprintf(stderr, "\r%s", e.String())
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
		fmt.Fprintf(stderr, "\nError: %v\n", err)
		return 1
	}

	if *progressFlag {
		fmt.Fprintln(stderr)
	}

	fmt.Fprintf(stdout, "Conversion complete: %s\n", *output)
	fmt.Fprintf(stdout, "  Source format: %s\n", result.SourceFormat)
	fmt.Fprintf(stdout, "  Target format: %s\n", result.TargetFormat)
	fmt.Fprintf(stdout, "  Bytes read: %d\n", result.BytesRead)
	fmt.Fprintf(stdout, "  Bytes written: %d\n", result.BytesWritten)
	fmt.Fprintf(stdout, "  Chunks processed: %d\n", result.ChunksRead)
	if result.ChunksSkipped > 0 {
		fmt.Fprintf(stdout, "  Chunks skipped: %d\n", result.ChunksSkipped)
	}

	cli.RenderDiagnostics(stdout, result.Warnings)

	return 0
}
