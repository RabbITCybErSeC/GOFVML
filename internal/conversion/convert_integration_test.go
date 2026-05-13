package conversion

import (
	"bytes"
	"context"
	"testing"

	"github.com/RabbITCybErSeC/gofvml/internal/image"
)

func TestConvert_RoundTrip_LiMEToRawAndBack(t *testing.T) {
	// Create a LiME image with known content.
	var limeImage bytes.Buffer
	
	// Write two LiME blocks.
	block1 := make([]byte, 1024)
	for i := range block1 {
		block1[i] = byte(i % 256)
	}
	h1 := image.NewLiMEHeader(0x1000, 0x1400)
	if err := h1.Encode(&limeImage); err != nil {
		t.Fatal(err)
	}
	limeImage.Write(block1)

	block2 := make([]byte, 512)
	for i := range block2 {
		block2[i] = byte((i + 1) % 256)
	}
	h2 := image.NewLiMEHeader(0x2000, 0x2200)
	if err := h2.Encode(&limeImage); err != nil {
		t.Fatal(err)
	}
	limeImage.Write(block2)

	// Convert LiME to raw.
	var rawImage bytes.Buffer
	opts := Options{
		SourceFormat:   FormatLiME,
		TargetFormat:   FormatRaw,
		SkipZeroChunks: false,
	}

	result, err := Convert(context.Background(), bytes.NewReader(limeImage.Bytes()), &rawImage, opts)
	if err != nil {
		t.Fatalf("LiME to raw conversion failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.BytesRead == 0 {
		t.Error("expected bytes read > 0")
	}

	// The raw image should have zeros before the first block, then block1, 
	// then zeros between blocks, then block2.
	rawData := rawImage.Bytes()
	
	// Check block1 content at offset 0x1000.
	for i := 0; i < len(block1); i++ {
		if rawData[0x1000+i] != block1[i] {
			t.Fatalf("block1 mismatch at offset %d", 0x1000+i)
		}
	}

	// Check block2 content at offset 0x2000.
	for i := 0; i < len(block2); i++ {
		if rawData[0x2000+i] != block2[i] {
			t.Fatalf("block2 mismatch at offset %d", 0x2000+i)
		}
	}
}

func TestConvert_RoundTrip_AVMLToRawAndBack(t *testing.T) {
	// Create an AVML image with known content.
	var avmlImage bytes.Buffer

	block1 := make([]byte, 1024)
	for i := range block1 {
		block1[i] = byte(i % 256)
	}
	h1 := image.NewAVMLHeader(0x1000, 0x1400)
	if err := image.EncodeAVMLBlock(&avmlImage, h1, block1); err != nil {
		t.Fatal(err)
	}

	// Convert AVML to raw.
	var rawImage bytes.Buffer
	opts := Options{
		SourceFormat:   FormatAVML,
		TargetFormat:   FormatRaw,
		SkipZeroChunks: false,
	}

	result, err := Convert(context.Background(), bytes.NewReader(avmlImage.Bytes()), &rawImage, opts)
	if err != nil {
		t.Fatalf("AVML to raw conversion failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}

	// Check block1 content at offset 0x1000.
	rawData := rawImage.Bytes()
	for i := 0; i < len(block1); i++ {
		if rawData[0x1000+i] != block1[i] {
			t.Fatalf("block1 mismatch at offset %d", 0x1000+i)
		}
	}

	// Convert raw back to AVML.
	var avmlImage2 bytes.Buffer
	opts2 := Options{
		SourceFormat:   FormatRaw,
		TargetFormat:   FormatAVML,
		SkipZeroChunks: false,
	}

	result2, err := Convert(context.Background(), bytes.NewReader(rawData), &avmlImage2, opts2)
	if err != nil {
		t.Fatalf("raw to AVML conversion failed: %v", err)
	}
	if !result2.Success {
		t.Error("expected success")
	}
}

func TestConvert_SameFormat(t *testing.T) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	opts := Options{
		SourceFormat: FormatRaw,
		TargetFormat: FormatRaw,
	}

	var out bytes.Buffer
	_, err := Convert(context.Background(), bytes.NewReader(data), &out, opts)
	if err == nil {
		t.Fatal("expected error for same-format conversion")
	}
}

func TestConvert_ZeroBlockSkipping(t *testing.T) {
	// Create raw data with a zero block in the middle.
	data := make([]byte, 4096)
	for i := 0; i < 1024; i++ {
		data[i] = byte(i % 256)
	}
	// 1024-2047 is zeros (skipped)
	for i := 2048; i < 3072; i++ {
		data[i] = byte((i + 1) % 256)
	}
	// 3072-4095 is zeros (skipped)

	var limeImage bytes.Buffer
	opts := Options{
		SourceFormat:   FormatRaw,
		TargetFormat:   FormatLiME,
		SkipZeroChunks: true,
		ChunkSize:      1024,
	}

	result, err := Convert(context.Background(), bytes.NewReader(data), &limeImage, opts)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.ChunksSkipped != 2 {
		t.Errorf("expected 2 skipped chunks, got %d", result.ChunksSkipped)
	}
	if result.ChunksRead != 4 {
		t.Errorf("expected 4 total chunks, got %d", result.ChunksRead)
	}
}
