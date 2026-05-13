package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/RabbITCybErSeC/gofvml/internal/acquisition"
	"github.com/RabbITCybErSeC/gofvml/internal/cli"
	"github.com/RabbITCybErSeC/gofvml/internal/conversion"
	"github.com/RabbITCybErSeC/gofvml/internal/iomem"
	"github.com/RabbITCybErSeC/gofvml/internal/process"
	"github.com/RabbITCybErSeC/gofvml/internal/procfs"
	"github.com/RabbITCybErSeC/gofvml/internal/progress"
	"github.com/RabbITCybErSeC/gofvml/internal/upload"
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
	case "convert":
		os.Exit(runConvert(os.Args[2:]))
	case "compress":
		os.Exit(runCompress(os.Args[2:]))
	case "upload":
		os.Exit(runUpload(os.Args[2:]))
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
	fmt.Println("  convert    Convert memory image formats")
	fmt.Println("  compress   Compress memory images to AVML-compatible format")
	fmt.Println("  upload     Upload memory images via HTTP PUT")
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

	meta, blocks := buildProcessArtifact(result, *strict, time.Now())

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

func buildProcessArtifact(result *process.Result, strict bool, timestamp time.Time) (process.ArtifactMetadata, []process.PayloadBlock) {
	meta := process.ArtifactMetadata{
		PID:        result.PID,
		Timestamp:  timestamp,
		Strict:     strict,
		BytesRead:  result.BytesRead,
		Mappings:   make([]procfs.Mapping, 0, len(result.Mappings)),
		ReadEvents: make([]process.ReadEvent, 0),
	}

	var blocks []process.PayloadBlock
	for i, m := range result.Mappings {
		meta.Mappings = append(meta.Mappings, m.Mapping)
		meta.ReadEvents = append(meta.ReadEvents, m.Events...)
		for _, block := range m.Blocks {
			block.MappingIndex = uint32(i)
			blocks = append(blocks, block)
		}
	}

	return meta, blocks
}

type convertConfig struct {
	input        string
	output       string
	fromFormat   string
	toFormat     string
	skipZero     bool
	progressFlag bool
}

type compressConfig struct {
	input        string
	output       string
	fromFormat   string
	format       string
	skipZero     bool
	progressFlag bool
}

type convertRunner interface {
	Run(context.Context, io.Reader, io.Writer, conversion.Options) (*conversion.Result, error)
}

type convertRunnerFunc func(context.Context, io.Reader, io.Writer, conversion.Options) (*conversion.Result, error)

func (f convertRunnerFunc) Run(ctx context.Context, input io.Reader, output io.Writer, opts conversion.Options) (*conversion.Result, error) {
	return f(ctx, input, output, opts)
}

var defaultConvertRunner convertRunner = convertRunnerFunc(conversion.Convert)

func runConvert(args []string) int {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	cfg := convertConfig{skipZero: true}
	fs.StringVar(&cfg.input, "input", "", "Input file path (required)")
	fs.StringVar(&cfg.output, "output", "", "Output file path (required)")
	fs.StringVar(&cfg.fromFormat, "from", "", "Source format: raw, lime, or avml (auto-detect if empty)")
	fs.StringVar(&cfg.toFormat, "to", "", "Target format: raw, lime, or avml (required)")
	fs.BoolVar(&cfg.skipZero, "skip-zero", true, "Skip all-zero chunks")
	fs.BoolVar(&cfg.progressFlag, "progress", false, "Show progress on stderr")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gofvml convert [options]\n\n")
		fmt.Fprintf(os.Stderr, "Convert between memory image formats.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return runConvertConfig(context.Background(), cfg, defaultConvertRunner)
}

func runCompress(args []string) int {
	fs := flag.NewFlagSet("compress", flag.ContinueOnError)
	cfg := compressConfig{format: "avml", skipZero: true}
	fs.StringVar(&cfg.input, "input", "", "Input file path (required)")
	fs.StringVar(&cfg.output, "output", "", "Output file path (required)")
	fs.StringVar(&cfg.fromFormat, "from", "", "Source format: raw, lime, or avml (auto-detect if empty)")
	fs.StringVar(&cfg.format, "format", "avml", "Compressed target format: avml")
	fs.BoolVar(&cfg.skipZero, "skip-zero", true, "Skip all-zero chunks")
	fs.BoolVar(&cfg.progressFlag, "progress", false, "Show progress on stderr")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gofvml compress [options]\n\n")
		fmt.Fprintf(os.Stderr, "Compress memory images to AVML-compatible format.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if cfg.input == "" || cfg.output == "" {
		fmt.Fprintln(os.Stderr, "Error: -input and -output are required")
		fs.Usage()
		return 1
	}
	opts, err := buildCompressOptions(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return runConvertFiles(context.Background(), cfg.input, cfg.output, opts, defaultConvertRunner)
}

func runConvertConfig(ctx context.Context, cfg convertConfig, runner convertRunner) int {
	if cfg.input == "" || cfg.output == "" || cfg.toFormat == "" {
		fmt.Fprintln(os.Stderr, "Error: -input, -output, and -to are required")
		return 1
	}
	sourceFormat, err := parseFormat(cfg.fromFormat, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	targetFormat, err := parseFormat(cfg.toFormat, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	opts := conversion.Options{
		SourceFormat:   sourceFormat,
		TargetFormat:   targetFormat,
		SkipZeroChunks: cfg.skipZero,
		Progress:       progressCallback(cfg.progressFlag),
	}
	return runConvertFiles(ctx, cfg.input, cfg.output, opts, runner)
}

func buildCompressOptions(cfg compressConfig) (conversion.Options, error) {
	targetFormat, err := parseFormat(defaultString(cfg.format, "avml"), false)
	if err != nil {
		return conversion.Options{}, err
	}
	if targetFormat != conversion.FormatAVML {
		return conversion.Options{}, fmt.Errorf("compress format must be avml")
	}
	sourceFormat, err := parseFormat(cfg.fromFormat, true)
	if err != nil {
		return conversion.Options{}, err
	}
	return conversion.Options{
		SourceFormat:   sourceFormat,
		TargetFormat:   targetFormat,
		SkipZeroChunks: cfg.skipZero,
		Progress:       progressCallback(cfg.progressFlag),
	}, nil
}

func runConvertFiles(ctx context.Context, inputPath, outputPath string, opts conversion.Options, runner convertRunner) int {
	input, err := os.Open(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot open input: %v\n", err)
		return 1
	}
	defer input.Close()

	output, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot create output: %v\n", err)
		return 1
	}
	defer output.Close()

	if err := runConversion(ctx, input, output, opts, runner); err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		return 1
	}
	return 0
}

func runConversion(ctx context.Context, input io.Reader, output io.Writer, opts conversion.Options, runner convertRunner) error {
	result, err := runner.Run(ctx, input, output, opts)
	if err != nil {
		return err
	}
	fmt.Printf("Conversion complete\n")
	fmt.Printf("  Source format: %s\n", result.SourceFormat)
	fmt.Printf("  Target format: %s\n", result.TargetFormat)
	fmt.Printf("  Bytes read: %d\n", result.BytesRead)
	fmt.Printf("  Bytes written: %d\n", result.BytesWritten)
	fmt.Printf("  Chunks processed: %d\n", result.ChunksRead)
	if result.ChunksSkipped > 0 {
		fmt.Printf("  Chunks skipped: %d\n", result.ChunksSkipped)
	}
	cli.RenderDiagnostics(os.Stdout, result.Warnings)
	return nil
}

type uploadRunner interface {
	Run(context.Context, upload.Options) (*upload.Result, error)
}

type uploadRunnerFunc func(context.Context, upload.Options) (*upload.Result, error)

func (f uploadRunnerFunc) Run(ctx context.Context, opts upload.Options) (*upload.Result, error) {
	return f(ctx, opts)
}

var defaultUploadRunner uploadRunner = uploadRunnerFunc(upload.Upload)

func runUpload(args []string) int {
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	var opts upload.Options
	fs.StringVar(&opts.FilePath, "file", "", "Local file path to upload (required)")
	fs.StringVar(&opts.URL, "url", "", "Destination URL for HTTP PUT (required)")
	fs.BoolVar(&opts.DeleteAfter, "delete-after", false, "Delete local file after successful upload")
	progressFlag := fs.Bool("progress", false, "Show progress on stderr")
	fs.IntVar(&opts.Retries, "retries", 3, "Number of retry attempts")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gofvml upload [options]\n\n")
		fmt.Fprintf(os.Stderr, "Upload memory images via HTTP PUT.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if opts.FilePath == "" || opts.URL == "" {
		fmt.Fprintln(os.Stderr, "Error: -file and -url are required")
		fs.Usage()
		return 1
	}
	opts.Progress = progressCallback(*progressFlag)
	if err := runUploadWorkflow(context.Background(), opts, defaultUploadRunner); err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		return 1
	}
	return 0
}

func runUploadWorkflow(ctx context.Context, opts upload.Options, runner uploadRunner) error {
	result, err := runner.Run(ctx, opts)
	if err != nil {
		if result != nil {
			cli.RenderDiagnostics(os.Stderr, result.Warnings)
		}
		return err
	}
	fmt.Printf("Upload complete: %s\n", opts.FilePath)
	fmt.Printf("  Destination: %s\n", result.URL)
	fmt.Printf("  Bytes uploaded: %d\n", result.BytesUploaded)
	cli.RenderDiagnostics(os.Stdout, result.Warnings)
	return nil
}

func parseFormat(value string, allowUnknown bool) (conversion.Format, error) {
	switch strings.ToLower(value) {
	case "":
		if allowUnknown {
			return conversion.FormatUnknown, nil
		}
	case "raw":
		return conversion.FormatRaw, nil
	case "lime":
		return conversion.FormatLiME, nil
	case "avml":
		return conversion.FormatAVML, nil
	}
	return conversion.FormatUnknown, fmt.Errorf("unknown format %q", value)
}

func progressCallback(enabled bool) progress.Callback {
	if !enabled {
		return nil
	}
	return func(e progress.Event) {
		fmt.Fprintf(os.Stderr, "\r%s", e.String())
	}
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
