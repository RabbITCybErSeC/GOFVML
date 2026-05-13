package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/RabbITCybErSeC/gofvml/internal/acquisition"
	"github.com/RabbITCybErSeC/gofvml/internal/cli"
	"github.com/RabbITCybErSeC/gofvml/internal/iomem"
	"github.com/RabbITCybErSeC/gofvml/internal/process"
	"github.com/RabbITCybErSeC/gofvml/internal/procfs"
	"github.com/RabbITCybErSeC/gofvml/internal/progress"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "physical":
		os.Exit(runPhysical(os.Args[2:]))
	case "process":
		os.Exit(runProcess(os.Args[2:]))
	case "version":
		fmt.Println("gofvml v0.1.0")
		os.Exit(0)
	case "help", "-h", "--help":
		printUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("gofvml - Linux volatile memory acquisition tool")
	fmt.Println()
	fmt.Println("Usage: gofvml <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  physical   Acquire physical memory")
	fmt.Println("  process    Acquire process memory")
	fmt.Println("  version    Print version")
	fmt.Println("  help       Show this help message")
	fmt.Println()
	fmt.Println("Run 'gofvml <command> -help' for command-specific options.")
}

func runPhysical(args []string) int {
	fs := flag.NewFlagSet("physical", flag.ExitOnError)
	output := fs.String("output", "", "Output file path (required)")
	format := fs.String("format", "lime", "Output format: lime or avml")
	source := fs.String("source", "", "Explicit source: crash, kcore, or mem (auto-detect if empty)")
	skipZero := fs.Bool("skip-zero", false, "Skip all-zero blocks")
	progressFlag := fs.Bool("progress", false, "Show progress on stderr")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if *output == "" {
		fmt.Fprintln(os.Stderr, "Error: -output is required")
		fs.Usage()
		return 1
	}

	*format = strings.ToLower(*format)
	if *format != "lime" && *format != "avml" {
		fmt.Fprintf(os.Stderr, "Error: unsupported format %q (use lime or avml)\n", *format)
		return 1
	}

	ctx := context.Background()

	// Discover physical memory ranges.
	ranges, err := iomem.ParseFile("/proc/iomem")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if len(ranges) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no System RAM ranges found in /proc/iomem")
		return 1
	}

	// Set up progress callback.
	var progressCb progress.Callback
	if *progressFlag {
		progressCb = func(e progress.Event) {
			fmt.Fprintf(os.Stderr, "\r%s", e.String())
		}
	}

	opts := acquisition.Options{
		OutputPath:     *output,
		Format:         *format,
		SourceName:     *source,
		Ranges:         ranges,
		Progress:       progressCb,
		SkipZeroBlocks: *skipZero,
	}

	result, err := acquisition.Acquire(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		if result != nil {
			cli.RenderDiagnostics(os.Stderr, result.Warnings)
		}
		return 1
	}

	if *progressFlag {
		fmt.Fprintln(os.Stderr)
	}

	fmt.Printf("Acquisition complete: %s\n", *output)
	fmt.Printf("  Source: %s\n", result.SourceName)
	fmt.Printf("  Blocks written: %d\n", result.BlocksWritten)
	fmt.Printf("  Bytes written: %d\n", result.BytesWritten)
	if result.BlocksSkipped > 0 {
		fmt.Printf("  Blocks skipped (zero): %d\n", result.BlocksSkipped)
	}

	cli.RenderDiagnostics(os.Stdout, result.Warnings)

	if len(result.Warnings) > 0 {
		return 0 // partial success
	}
	return 0
}

func runProcess(args []string) int {
	fs := flag.NewFlagSet("process", flag.ExitOnError)
	pid := fs.Int("pid", 0, "Target process ID (required)")
	output := fs.String("output", "", "Output artifact path (required)")
	strict := fs.Bool("strict", false, "Fail on first mapping error")
	maxBytes := fs.Uint64("max-bytes", 0, "Maximum total bytes to read (0 = unlimited)")
	pathnameMatch := fs.String("pathname", "", "Filter mappings by pathname substring")
	progressFlag := fs.Bool("progress", false, "Show progress on stderr")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if *pid <= 0 {
		fmt.Fprintln(os.Stderr, "Error: -pid is required and must be positive")
		fs.Usage()
		return 1
	}
	if *output == "" {
		fmt.Fprintln(os.Stderr, "Error: -output is required")
		fs.Usage()
		return 1
	}

	ctx := context.Background()

	filter := process.DefaultFilter()
	if *maxBytes > 0 {
		filter.MaxBytes = *maxBytes
	}
	if *pathnameMatch != "" {
		filter.PathnameMatch = *pathnameMatch
	}

	var progressCb progress.Callback
	if *progressFlag {
		progressCb = func(e progress.Event) {
			fmt.Fprintf(os.Stderr, "\r%s", e.String())
		}
	}

	opts := process.Options{
		PID:      *pid,
		Filter:   filter,
		Strict:   *strict,
		Progress: progressCb,
	}

	result, err := process.Acquire(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		if result != nil {
			cli.RenderDiagnostics(os.Stderr, result.Warnings)
		}
		return 1
	}

	if *progressFlag {
		fmt.Fprintln(os.Stderr)
	}

	// Write artifact.
	artifactFile, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating artifact: %v\n", err)
		return 1
	}
	defer artifactFile.Close()

	// Build artifact metadata and blocks from result.
	meta := process.ArtifactMetadata{
		PID:        result.PID,
		Timestamp:  time.Now(),
		Strict:     *strict,
		BytesRead:  result.BytesRead,
		Mappings:   make([]procfs.Mapping, 0, len(result.Mappings)),
		ReadEvents: make([]process.ReadEvent, 0),
	}

	var blocks []process.PayloadBlock
	for _, m := range result.Mappings {
		meta.Mappings = append(meta.Mappings, m.Mapping)
		meta.ReadEvents = append(meta.ReadEvents, m.Events...)
		// Create a placeholder block for each mapping.
		blocks = append(blocks, process.PayloadBlock{
			VirtualAddress:  m.Mapping.Start,
			MappingIndex:    uint32(len(meta.Mappings) - 1),
			CompressionType: process.CompressionNone,
			Status:          process.StatusOK,
		})
	}

	if err := process.WriteArtifact(artifactFile, blocks, meta); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing artifact: %v\n", err)
		return 1
	}

	fmt.Printf("Process acquisition complete: %s\n", *output)
	fmt.Printf("  PID: %d\n", result.PID)
	fmt.Printf("  Mappings: %d\n", len(result.Mappings))
	fmt.Printf("  Bytes read: %d\n", result.BytesRead)
	fmt.Printf("  Artifact blocks: %d\n", len(blocks))

	cli.RenderDiagnostics(os.Stdout, result.Warnings)

	return 0
}
