package conversion

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/RabbITCybErSeC/gofvml/internal/image"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want Format
	}{
		{
			name: "lime",
			data: func() []byte {
				var buf bytes.Buffer
				h := image.NewLiMEHeader(0x1000, 0x2000)
				h.Encode(&buf)
				return buf.Bytes()
			}(),
			want: FormatLiME,
		},
		{
			name: "avml",
			data: func() []byte {
				var buf bytes.Buffer
				h := image.NewAVMLHeader(0x1000, 0x1003)
				image.EncodeAVMLBlock(&buf, h, []byte{1, 2, 3})
				return buf.Bytes()
			}(),
			want: FormatAVML,
		},
		{
			name: "raw",
			data: []byte{0, 1, 2, 3, 4, 5, 6, 7},
			want: FormatRaw,
		},
		{
			name: "empty",
			data: []byte{},
			want: FormatUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectFormat(bytes.NewReader(tt.data))
			if tt.name == "empty" {
				if err == nil {
					t.Error("expected error for empty input")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("DetectFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateFormatPair(t *testing.T) {
	tests := []struct {
		source  Format
		target  Format
		wantErr bool
	}{
		{FormatRaw, FormatLiME, false},
		{FormatLiME, FormatRaw, false},
		{FormatRaw, FormatAVML, false},
		{FormatAVML, FormatRaw, false},
		{FormatLiME, FormatAVML, false},
		{FormatAVML, FormatLiME, false},
		{FormatRaw, FormatRaw, true},
		{FormatLiME, FormatLiME, true},
		{FormatAVML, FormatAVML, true},
		{FormatUnknown, FormatRaw, true},
		{FormatRaw, FormatUnknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.source.String()+"_to_"+tt.target.String(), func(t *testing.T) {
			err := ValidateFormatPair(tt.source, tt.target)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConvertRawToLiME(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	input := bytes.NewReader(data)
	var output bytes.Buffer

	ctx := context.Background()
	opts := Options{
		SourceFormat:   FormatRaw,
		TargetFormat:   FormatLiME,
		ChunkSize:      4,
		SkipZeroChunks: false,
	}

	result, err := Convert(ctx, input, &output, opts)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.ChunksRead != 2 {
		t.Errorf("ChunksRead = %d, want 2", result.ChunksRead)
	}

	// Decode and verify.
	readData := output.Bytes()
	header1, err := image.DecodeLiMEHeader(bytes.NewReader(readData))
	if err != nil {
		t.Fatalf("DecodeLiMEHeader error: %v", err)
	}
	if header1.Start != 0 {
		t.Errorf("header1.Start = 0x%x, want 0x0", header1.Start)
	}
	if header1.ExclusiveEnd() != 4 {
		t.Errorf("header1.ExclusiveEnd = 0x%x, want 0x4", header1.ExclusiveEnd())
	}
}

func TestConvertRawToAVML(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	input := bytes.NewReader(data)
	var output bytes.Buffer

	ctx := context.Background()
	opts := Options{
		SourceFormat:   FormatRaw,
		TargetFormat:   FormatAVML,
		ChunkSize:      4,
		SkipZeroChunks: false,
	}

	result, err := Convert(ctx, input, &output, opts)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}

	// Decode and verify.
	readData := output.Bytes()
	header1, payload1, err := image.DecodeAVMLBlock(bytes.NewReader(readData))
	if err != nil {
		t.Fatalf("DecodeAVMLBlock error: %v", err)
	}
	if header1.Start != 0 {
		t.Errorf("header1.Start = 0x%x, want 0x0", header1.Start)
	}
	if !bytes.Equal(payload1, []byte{1, 2, 3, 4}) {
		t.Errorf("payload1 = %v, want [1 2 3 4]", payload1)
	}
}

func TestConvertLiMEToRaw(t *testing.T) {
	// Create LiME input.
	var input bytes.Buffer
	data1 := []byte{1, 2, 3, 4}
	h1 := image.NewLiMEHeader(0, uint64(len(data1)))
	h1.Encode(&input)
	input.Write(data1)

	data2 := []byte{5, 6, 7, 8}
	h2 := image.NewLiMEHeader(4, 4+uint64(len(data2)))
	h2.Encode(&input)
	input.Write(data2)

	var output bytes.Buffer
	ctx := context.Background()
	opts := Options{
		SourceFormat: FormatLiME,
		TargetFormat: FormatRaw,
	}

	result, err := Convert(ctx, &input, &output, opts)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}

	got := output.Bytes()
	want := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	if !bytes.Equal(got, want) {
		t.Errorf("output = %v, want %v", got, want)
	}
}

func TestConvertAVMLToRaw(t *testing.T) {
	// Create AVML input.
	var input bytes.Buffer
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	h := image.NewAVMLHeader(0, uint64(len(data)))
	image.EncodeAVMLBlock(&input, h, data)

	var output bytes.Buffer
	ctx := context.Background()
	opts := Options{
		SourceFormat: FormatAVML,
		TargetFormat: FormatRaw,
	}

	result, err := Convert(ctx, &input, &output, opts)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}

	got := output.Bytes()
	if !bytes.Equal(got, data) {
		t.Errorf("output = %v, want %v", got, data)
	}
}

func TestConvertLiMEToRawZeroFillsSparseGaps(t *testing.T) {
	var input bytes.Buffer
	data1 := []byte{1, 2}
	h1 := image.NewLiMEHeader(4, 6)
	h1.Encode(&input)
	input.Write(data1)

	data2 := []byte{3, 4}
	h2 := image.NewLiMEHeader(8, 10)
	h2.Encode(&input)
	input.Write(data2)

	var output bytes.Buffer
	result, err := Convert(context.Background(), &input, &output, Options{
		SourceFormat: FormatLiME,
		TargetFormat: FormatRaw,
	})
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}

	want := []byte{0, 0, 0, 0, 1, 2, 0, 0, 3, 4}
	if got := output.Bytes(); !bytes.Equal(got, want) {
		t.Fatalf("output = %v, want %v", got, want)
	}
}

func TestConvertAVMLToRawZeroFillsSparseGaps(t *testing.T) {
	var input bytes.Buffer
	h := image.NewAVMLHeader(4, 6)
	if err := image.EncodeAVMLBlock(&input, h, []byte{1, 2}); err != nil {
		t.Fatalf("EncodeAVMLBlock error: %v", err)
	}

	var output bytes.Buffer
	result, err := Convert(context.Background(), &input, &output, Options{
		SourceFormat: FormatAVML,
		TargetFormat: FormatRaw,
	})
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}

	want := []byte{0, 0, 0, 0, 1, 2}
	if got := output.Bytes(); !bytes.Equal(got, want) {
		t.Fatalf("output = %v, want %v", got, want)
	}
}

func TestConvertEncodedToRawDoesNotInventTrailingSkippedZeroExtent(t *testing.T) {
	var input bytes.Buffer
	h := image.NewLiMEHeader(0, 2)
	h.Encode(&input)
	input.Write([]byte{1, 2})

	var output bytes.Buffer
	_, err := Convert(context.Background(), &input, &output, Options{
		SourceFormat: FormatLiME,
		TargetFormat: FormatRaw,
	})
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}

	if got, want := output.Bytes(), []byte{1, 2}; !bytes.Equal(got, want) {
		t.Fatalf("output = %v, want %v", got, want)
	}
}

func TestConvertLiMEToAVML(t *testing.T) {
	// Create LiME input.
	var input bytes.Buffer
	data := []byte{1, 2, 3, 4}
	h := image.NewLiMEHeader(0, uint64(len(data)))
	h.Encode(&input)
	input.Write(data)

	var output bytes.Buffer
	ctx := context.Background()
	opts := Options{
		SourceFormat: FormatLiME,
		TargetFormat: FormatAVML,
	}

	result, err := Convert(ctx, &input, &output, opts)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}

	// Decode AVML output.
	header, payload, err := image.DecodeAVMLBlock(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatalf("DecodeAVMLBlock error: %v", err)
	}
	if header.Start != 0 {
		t.Errorf("Start = 0x%x, want 0x0", header.Start)
	}
	if !bytes.Equal(payload, data) {
		t.Errorf("payload = %v, want %v", payload, data)
	}
}

func TestConvertRawToAVMLSplitsLargeChunks(t *testing.T) {
	data := bytes.Repeat([]byte{0x5a}, image.AVMLBlockSize+3)
	var output bytes.Buffer

	result, err := Convert(context.Background(), bytes.NewReader(data), &output, Options{
		SourceFormat: FormatRaw,
		TargetFormat: FormatAVML,
		ChunkSize:    int64(len(data)),
	})
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}

	blocks := decodeAVMLBlocks(t, output.Bytes())
	if len(blocks) != 2 {
		t.Fatalf("block count = %d, want 2", len(blocks))
	}
	if blocks[0].header.Start != 0 || blocks[0].header.RangeLen() != image.AVMLBlockSize {
		t.Fatalf("first block range = 0x%x len %d, want start 0 len %d", blocks[0].header.Start, blocks[0].header.RangeLen(), image.AVMLBlockSize)
	}
	if blocks[1].header.Start != image.AVMLBlockSize || blocks[1].header.RangeLen() != 3 {
		t.Fatalf("second block range = 0x%x len %d, want start %d len 3", blocks[1].header.Start, blocks[1].header.RangeLen(), image.AVMLBlockSize)
	}
}

func TestConvertLiMEToAVMLSplitsLargeBlocks(t *testing.T) {
	data := bytes.Repeat([]byte{0xa5}, image.AVMLBlockSize+1)
	var input bytes.Buffer
	h := image.NewLiMEHeader(0x1000, 0x1000+uint64(len(data)))
	h.Encode(&input)
	input.Write(data)

	var output bytes.Buffer
	_, err := Convert(context.Background(), &input, &output, Options{
		SourceFormat: FormatLiME,
		TargetFormat: FormatAVML,
	})
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}

	blocks := decodeAVMLBlocks(t, output.Bytes())
	if len(blocks) != 2 {
		t.Fatalf("block count = %d, want 2", len(blocks))
	}
	if blocks[0].header.RangeLen() != image.AVMLBlockSize {
		t.Fatalf("first block length = %d, want %d", blocks[0].header.RangeLen(), image.AVMLBlockSize)
	}
	if blocks[1].header.Start != 0x1000+image.AVMLBlockSize || blocks[1].header.RangeLen() != 1 {
		t.Fatalf("second block range = 0x%x len %d", blocks[1].header.Start, blocks[1].header.RangeLen())
	}
}

func TestConvertAVMLToLiME(t *testing.T) {
	// Create AVML input.
	var input bytes.Buffer
	data := []byte{1, 2, 3, 4}
	h := image.NewAVMLHeader(0, uint64(len(data)))
	image.EncodeAVMLBlock(&input, h, data)

	var output bytes.Buffer
	ctx := context.Background()
	opts := Options{
		SourceFormat: FormatAVML,
		TargetFormat: FormatLiME,
	}

	result, err := Convert(ctx, &input, &output, opts)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}

	// Decode LiME output.
	header, err := image.DecodeLiMEHeader(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatalf("DecodeLiMEHeader error: %v", err)
	}
	if header.Start != 0 {
		t.Errorf("Start = 0x%x, want 0x0", header.Start)
	}
}

func TestConvertSameFormat(t *testing.T) {
	ctx := context.Background()
	opts := Options{
		SourceFormat: FormatLiME,
		TargetFormat: FormatLiME,
	}

	_, err := Convert(ctx, bytes.NewReader([]byte{}), io.Discard, opts)
	if err == nil {
		t.Error("expected error for same-format conversion")
	}
}

func TestConvertZeroChunkSkipping(t *testing.T) {
	data := make([]byte, 8) // all zeros
	input := bytes.NewReader(data)
	var output bytes.Buffer

	ctx := context.Background()
	opts := Options{
		SourceFormat:   FormatRaw,
		TargetFormat:   FormatLiME,
		ChunkSize:      4,
		SkipZeroChunks: true,
	}

	result, err := Convert(ctx, input, &output, opts)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.ChunksSkipped != 2 {
		t.Errorf("ChunksSkipped = %d, want 2", result.ChunksSkipped)
	}
	// Output should be empty since all chunks were zero.
	if output.Len() != 0 {
		t.Errorf("output length = %d, want 0", output.Len())
	}
}

func TestConvertCancellation(t *testing.T) {
	// Large input to ensure cancellation happens mid-conversion.
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	input := bytes.NewReader(data)
	var output bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately.
	cancel()

	opts := Options{
		SourceFormat: FormatRaw,
		TargetFormat: FormatLiME,
		ChunkSize:    1024,
	}

	_, err := Convert(ctx, input, &output, opts)
	if err == nil {
		t.Error("expected error after cancellation")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestConvertAutoDetect(t *testing.T) {
	// Create LiME input without specifying source format.
	var input bytes.Buffer
	data := []byte{1, 2, 3, 4}
	h := image.NewLiMEHeader(0, uint64(len(data)))
	h.Encode(&input)
	input.Write(data)

	var output bytes.Buffer
	ctx := context.Background()
	opts := Options{
		SourceFormat: FormatUnknown, // auto-detect
		TargetFormat: FormatRaw,
	}

	result, err := Convert(ctx, &input, &output, opts)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if result.SourceFormat != FormatLiME {
		t.Errorf("SourceFormat = %v, want LiME", result.SourceFormat)
	}
}

type avmlTestBlock struct {
	header *image.AVMLHeader
	data   []byte
}

func decodeAVMLBlocks(t *testing.T, data []byte) []avmlTestBlock {
	t.Helper()
	r := bytes.NewReader(data)
	var blocks []avmlTestBlock
	for r.Len() > 0 {
		header, payload, err := image.DecodeAVMLBlock(r)
		if err != nil {
			t.Fatalf("DecodeAVMLBlock error: %v", err)
		}
		blocks = append(blocks, avmlTestBlock{header: header, data: payload})
	}
	return blocks
}
