package image

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestLiMEHeaderEncodeDecode(t *testing.T) {
	tests := []struct {
		name         string
		start        uint64
		exclusiveEnd uint64
	}{
		{"single_byte", 0x1000, 0x1001},
		{"small_range", 0x1000, 0x2000},
		{"large_range", 0x0, 0x100000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewLiMEHeader(tt.start, tt.exclusiveEnd)

			var buf bytes.Buffer
			if err := h.Encode(&buf); err != nil {
				t.Fatalf("Encode error: %v", err)
			}

			// Verify size.
			if buf.Len() != LiMEHeaderSize {
				t.Errorf("encoded size = %d, want %d", buf.Len(), LiMEHeaderSize)
			}

			// Decode and verify.
			decoded, err := DecodeLiMEHeader(&buf)
			if err != nil {
				t.Fatalf("Decode error: %v", err)
			}

			if decoded.Magic != LiMEMagic {
				t.Errorf("Magic = 0x%08x, want 0x%08x", decoded.Magic, LiMEMagic)
			}
			if decoded.Version != LiMEVersion {
				t.Errorf("Version = %d, want %d", decoded.Version, LiMEVersion)
			}
			if decoded.Start != tt.start {
				t.Errorf("Start = 0x%x, want 0x%x", decoded.Start, tt.start)
			}

			// Verify exclusive-end round-trip.
			if decoded.ExclusiveEnd() != tt.exclusiveEnd {
				t.Errorf("ExclusiveEnd = 0x%x, want 0x%x", decoded.ExclusiveEnd(), tt.exclusiveEnd)
			}
		})
	}
}

func TestLiMEHeaderInvalidMagic(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(0xDEADBEEF)) // bad magic
	binary.Write(&buf, binary.LittleEndian, uint32(LiMEVersion))
	binary.Write(&buf, binary.LittleEndian, uint64(0x1000))
	binary.Write(&buf, binary.LittleEndian, uint64(0x2000))
	buf.Write(make([]byte, 8)) // padding

	_, err := DecodeLiMEHeader(&buf)
	if err == nil {
		t.Error("expected error for invalid magic")
	}
}

func TestLiMEHeaderInvalidVersion(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(LiMEMagic))
	binary.Write(&buf, binary.LittleEndian, uint32(99)) // bad version
	binary.Write(&buf, binary.LittleEndian, uint64(0x1000))
	binary.Write(&buf, binary.LittleEndian, uint64(0x2000))
	buf.Write(make([]byte, 8)) // padding

	_, err := DecodeLiMEHeader(&buf)
	if err == nil {
		t.Error("expected error for invalid version")
	}
}

func TestLiMEHeaderNonZeroPadding(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(LiMEMagic))
	binary.Write(&buf, binary.LittleEndian, uint32(LiMEVersion))
	binary.Write(&buf, binary.LittleEndian, uint64(0x1000))
	binary.Write(&buf, binary.LittleEndian, uint64(0x2000))
	buf.Write([]byte{1, 0, 0, 0, 0, 0, 0, 0}) // non-zero padding

	_, err := DecodeLiMEHeader(&buf)
	if err == nil {
		t.Error("expected error for non-zero padding")
	}
}

func TestLiMEHeaderReversedRange(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(LiMEMagic))
	binary.Write(&buf, binary.LittleEndian, uint32(LiMEVersion))
	binary.Write(&buf, binary.LittleEndian, uint64(0x2000))
	binary.Write(&buf, binary.LittleEndian, uint64(0x1000)) // end < start
	buf.Write(make([]byte, 8))                              // padding

	_, err := DecodeLiMEHeader(&buf)
	if err == nil {
		t.Error("expected error for reversed range")
	}
}

func TestLiMEHeaderTruncated(t *testing.T) {
	buf := bytes.NewReader([]byte{1, 2, 3}) // too short
	_, err := DecodeLiMEHeader(buf)
	if err == nil {
		t.Error("expected error for truncated header")
	}
}

func TestLiMEHeaderExclusiveEndConversion(t *testing.T) {
	// Range [0x1000, 0x2000) -> LiME stores [0x1000, 0x1fff]
	h := NewLiMEHeader(0x1000, 0x2000)
	if h.End != 0x1fff {
		t.Errorf("End = 0x%x, want 0x1fff", h.End)
	}
	if h.ExclusiveEnd() != 0x2000 {
		t.Errorf("ExclusiveEnd = 0x%x, want 0x2000", h.ExclusiveEnd())
	}
	if h.RangeLen() != 0x1000 {
		t.Errorf("RangeLen = %d, want %d", h.RangeLen(), 0x1000)
	}
}

func TestAVMLHeaderEncodeDecode(t *testing.T) {
	tests := []struct {
		name         string
		start        uint64
		exclusiveEnd uint64
	}{
		{"single_byte", 0x1000, 0x1001},
		{"small_range", 0x1000, 0x2000},
		{"large_range", 0x0, 0x100000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewAVMLHeader(tt.start, tt.exclusiveEnd)

			var buf bytes.Buffer
			if err := h.Encode(&buf); err != nil {
				t.Fatalf("Encode error: %v", err)
			}

			if buf.Len() != AVMLHeaderSize {
				t.Errorf("encoded size = %d, want %d", buf.Len(), AVMLHeaderSize)
			}

			decoded, err := DecodeAVMLHeader(&buf)
			if err != nil {
				t.Fatalf("Decode error: %v", err)
			}

			if decoded.Magic != AVMLMagic {
				t.Errorf("Magic = 0x%08x, want 0x%08x", decoded.Magic, AVMLMagic)
			}
			if decoded.Version != AVMLVersion {
				t.Errorf("Version = %d, want %d", decoded.Version, AVMLVersion)
			}
			if decoded.Start != tt.start {
				t.Errorf("Start = 0x%x, want 0x%x", decoded.Start, tt.start)
			}
			if decoded.ExclusiveEnd() != tt.exclusiveEnd {
				t.Errorf("ExclusiveEnd = 0x%x, want 0x%x", decoded.ExclusiveEnd(), tt.exclusiveEnd)
			}
		})
	}
}

func TestAVMLBlockRoundTrip(t *testing.T) {
	data := []byte("hello, world! this is test data for AVML compression.")
	header := NewAVMLHeader(0x1000, 0x1000+uint64(len(data)))

	var buf bytes.Buffer
	if err := EncodeAVMLBlock(&buf, header, data); err != nil {
		t.Fatalf("EncodeAVMLBlock error: %v", err)
	}

	decodedHeader, decodedData, err := DecodeAVMLBlock(&buf)
	if err != nil {
		t.Fatalf("DecodeAVMLBlock error: %v", err)
	}

	if decodedHeader.Start != header.Start {
		t.Errorf("Start = 0x%x, want 0x%x", decodedHeader.Start, header.Start)
	}
	if decodedHeader.ExclusiveEnd() != header.ExclusiveEnd() {
		t.Errorf("ExclusiveEnd = 0x%x, want 0x%x", decodedHeader.ExclusiveEnd(), header.ExclusiveEnd())
	}
	if !bytes.Equal(decodedData, data) {
		t.Errorf("data mismatch: got %q, want %q", decodedData, data)
	}
}

func TestAVMLBlockLayoutUsesSnappyFrameAndEOFLengthTrailer(t *testing.T) {
	data := []byte("snappy framed payload")
	header := NewAVMLHeader(0x1000, 0x1000+uint64(len(data)))

	var buf bytes.Buffer
	if err := EncodeAVMLBlock(&buf, header, data); err != nil {
		t.Fatalf("EncodeAVMLBlock error: %v", err)
	}

	encoded := buf.Bytes()
	if len(encoded) <= AVMLHeaderSize+AVMLTrailerSize {
		t.Fatalf("encoded block too short: %d", len(encoded))
	}

	payload := encoded[AVMLHeaderSize : len(encoded)-AVMLTrailerSize]
	if !bytes.HasPrefix(payload, snappyStreamIdentifier) {
		t.Fatalf("payload does not start with snappy stream identifier: % x", payload[:min(len(payload), len(snappyStreamIdentifier))])
	}

	trailer := encoded[len(encoded)-AVMLTrailerSize:]
	gotLen := binary.LittleEndian.Uint64(trailer)
	if gotLen != uint64(len(payload)) {
		t.Fatalf("trailer length = %d, want compressed payload length %d", gotLen, len(payload))
	}
}

func TestAVMLBlockZeroData(t *testing.T) {
	data := make([]byte, 1024) // all zeros
	header := NewAVMLHeader(0x1000, 0x1000+uint64(len(data)))

	var buf bytes.Buffer
	if err := EncodeAVMLBlock(&buf, header, data); err != nil {
		t.Fatalf("EncodeAVMLBlock error: %v", err)
	}

	decodedHeader, decodedData, err := DecodeAVMLBlock(&buf)
	if err != nil {
		t.Fatalf("DecodeAVMLBlock error: %v", err)
	}

	if decodedHeader.Start != header.Start {
		t.Errorf("Start mismatch")
	}
	if len(decodedData) != len(data) {
		t.Errorf("data length = %d, want %d", len(decodedData), len(data))
	}
	for i, b := range decodedData {
		if b != 0 {
			t.Errorf("byte[%d] = %d, want 0", i, b)
		}
	}
}

func TestAVMLSplitRange(t *testing.T) {
	// Range of 20 MiB should split into two 16 MiB + 4 MiB blocks.
	start := uint64(0x1000)
	end := start + 20*1024*1024
	blocks := SplitRange(start, end)

	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	if blocks[0].Start != start {
		t.Errorf("block[0].Start = 0x%x, want 0x%x", blocks[0].Start, start)
	}
	if blocks[0].End != start+AVMLBlockSize {
		t.Errorf("block[0].End = 0x%x, want 0x%x", blocks[0].End, start+AVMLBlockSize)
	}
	if blocks[1].Start != start+AVMLBlockSize {
		t.Errorf("block[1].Start = 0x%x, want 0x%x", blocks[1].Start, start+AVMLBlockSize)
	}
	if blocks[1].End != end {
		t.Errorf("block[1].End = 0x%x, want 0x%x", blocks[1].End, end)
	}
}

func TestAVMLSplitRangeExact(t *testing.T) {
	// Exact 16 MiB range should produce exactly one block.
	start := uint64(0x1000)
	end := start + AVMLBlockSize
	blocks := SplitRange(start, end)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Start != start {
		t.Errorf("block[0].Start = 0x%x, want 0x%x", blocks[0].Start, start)
	}
	if blocks[0].End != end {
		t.Errorf("block[0].End = 0x%x, want 0x%x", blocks[0].End, end)
	}
}

func TestAVMLSplitRangeSmall(t *testing.T) {
	// Small range should produce one block.
	start := uint64(0x1000)
	end := uint64(0x2000)
	blocks := SplitRange(start, end)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Start != start {
		t.Errorf("block[0].Start = 0x%x, want 0x%x", blocks[0].Start, start)
	}
	if blocks[0].End != end {
		t.Errorf("block[0].End = 0x%x, want 0x%x", blocks[0].End, end)
	}
}

func TestAVMLDecodeTruncatedHeader(t *testing.T) {
	buf := bytes.NewReader([]byte{1, 2, 3})
	_, err := DecodeAVMLHeader(buf)
	if err == nil {
		t.Error("expected error for truncated header")
	}
}

func TestAVMLDecodeTruncatedTrailer(t *testing.T) {
	var buf bytes.Buffer
	h := NewAVMLHeader(0x1000, 0x2000)
	h.Encode(&buf)
	// Write only partial trailer (less than 8 bytes).
	buf.Write([]byte{1, 2, 3})

	_, _, err := DecodeAVMLBlock(&buf)
	if err == nil {
		t.Error("expected error for truncated trailer")
	}
}

func TestAVMLDecodeTruncatedPayload(t *testing.T) {
	var buf bytes.Buffer
	h := NewAVMLHeader(0x1000, 0x1004)
	h.Encode(&buf)
	buf.Write(snappyStreamIdentifier)
	buf.Write([]byte{0x00, 0x08, 0x00, 0x00}) // chunk header without payload

	_, _, err := DecodeAVMLBlock(&buf)
	if err == nil {
		t.Error("expected error for truncated payload")
	}
}

func TestAVMLDecodeInvalidSnappy(t *testing.T) {
	var buf bytes.Buffer
	h := NewAVMLHeader(0x1000, 0x2000)
	h.Encode(&buf)
	buf.Write([]byte{0xFF, 0xFF, 0xFF, 0xFF})

	_, _, err := DecodeAVMLBlock(&buf)
	if err == nil {
		t.Error("expected error for invalid snappy data")
	}
}

func TestAVMLDecodeRejectsMismatchedTrailerLength(t *testing.T) {
	data := []byte("trailer mismatch")
	header := NewAVMLHeader(0x1000, 0x1000+uint64(len(data)))

	var buf bytes.Buffer
	if err := EncodeAVMLBlock(&buf, header, data); err != nil {
		t.Fatalf("EncodeAVMLBlock error: %v", err)
	}

	encoded := buf.Bytes()
	binary.LittleEndian.PutUint64(encoded[len(encoded)-AVMLTrailerSize:], 1)

	_, _, err := DecodeAVMLBlock(bytes.NewReader(encoded))
	if err == nil {
		t.Error("expected error for mismatched trailer length")
	}
}

func TestAVMLInvalidMagic(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(0xDEADBEEF))
	binary.Write(&buf, binary.LittleEndian, uint32(AVMLVersion))
	binary.Write(&buf, binary.LittleEndian, uint64(0x1000))
	binary.Write(&buf, binary.LittleEndian, uint64(0x2000))
	buf.Write(make([]byte, 8))

	_, err := DecodeAVMLHeader(&buf)
	if err == nil {
		t.Error("expected error for invalid magic")
	}
}

func TestAVMLInvalidVersion(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(AVMLMagic))
	binary.Write(&buf, binary.LittleEndian, uint32(99))
	binary.Write(&buf, binary.LittleEndian, uint64(0x1000))
	binary.Write(&buf, binary.LittleEndian, uint64(0x2000))
	buf.Write(make([]byte, 8))

	_, err := DecodeAVMLHeader(&buf)
	if err == nil {
		t.Error("expected error for invalid version")
	}
}

func TestAVMLReversedRange(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(AVMLMagic))
	binary.Write(&buf, binary.LittleEndian, uint32(AVMLVersion))
	binary.Write(&buf, binary.LittleEndian, uint64(0x2000))
	binary.Write(&buf, binary.LittleEndian, uint64(0x1000))
	buf.Write(make([]byte, 8))

	_, err := DecodeAVMLHeader(&buf)
	if err == nil {
		t.Error("expected error for reversed range")
	}
}

func TestIsAllZeros(t *testing.T) {
	tests := []struct {
		data []byte
		want bool
	}{
		{[]byte{}, true},
		{[]byte{0, 0, 0}, true},
		{[]byte{0, 1, 0}, false},
		{[]byte{1}, false},
		{make([]byte, 4096), true},
	}

	for _, tt := range tests {
		if got := isAllZeros(tt.data); got != tt.want {
			t.Errorf("isAllZeros(%v) = %v, want %v", tt.data, got, tt.want)
		}
	}
}

func isAllZeros(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}
