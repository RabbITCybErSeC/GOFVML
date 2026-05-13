package image

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/golang/snappy"
)

// AVMLHeaderSize is the size of an AVML-compressed header in bytes.
const AVMLHeaderSize = 32

// AVMLMagic is the AVML magic number.
const AVMLMagic uint32 = 0x41564d4c // "AVML"

// AVMLVersion is the supported AVML-compressed version.
const AVMLVersion uint32 = 2

// AVMLBlockSize is the maximum size of an AVML-compressed block (16 MiB).
const AVMLBlockSize = 16 * 1024 * 1024

// AVMLHeader represents an AVML-compressed block header.
type AVMLHeader struct {
	Magic   uint32
	Version uint32
	Start   uint64 // physical start address
	End     uint64 // physical end address (inclusive in AVML format)
}

// Encode writes the AVML header to w in little-endian format.
func (h *AVMLHeader) Encode(w io.Writer) error {
	var buf [AVMLHeaderSize]byte
	binary.LittleEndian.PutUint32(buf[0:4], h.Magic)
	binary.LittleEndian.PutUint32(buf[4:8], h.Version)
	binary.LittleEndian.PutUint64(buf[8:16], h.Start)
	binary.LittleEndian.PutUint64(buf[16:24], h.End)
	// buf[24:32] is reserved (zero padding)
	_, err := w.Write(buf[:])
	return err
}

// DecodeAVMLHeader reads and decodes an AVML header from r.
func DecodeAVMLHeader(r io.Reader) (*AVMLHeader, error) {
	var buf [AVMLHeaderSize]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return nil, fmt.Errorf("read AVML header: %w", err)
	}

	h := &AVMLHeader{
		Magic:   binary.LittleEndian.Uint32(buf[0:4]),
		Version: binary.LittleEndian.Uint32(buf[4:8]),
		Start:   binary.LittleEndian.Uint64(buf[8:16]),
		End:     binary.LittleEndian.Uint64(buf[16:24]),
	}

	if h.Magic != AVMLMagic {
		return nil, fmt.Errorf("invalid AVML magic: got 0x%08x, want 0x%08x", h.Magic, AVMLMagic)
	}

	if h.Version != AVMLVersion {
		return nil, fmt.Errorf("unsupported AVML version: got %d, want %d", h.Version, AVMLVersion)
	}

	// Validate reserved padding is zero.
	for i := 24; i < AVMLHeaderSize; i++ {
		if buf[i] != 0 {
			return nil, fmt.Errorf("non-zero padding at byte %d", i)
		}
	}

	if h.End < h.Start {
		return nil, fmt.Errorf("invalid range: end (0x%x) < start (0x%x)", h.End, h.Start)
	}

	return h, nil
}

// NewAVMLHeader creates an AVML header from a physical range.
func NewAVMLHeader(start, exclusiveEnd uint64) *AVMLHeader {
	inclusiveEnd := exclusiveEnd
	if exclusiveEnd > start {
		inclusiveEnd = exclusiveEnd - 1
	}

	return &AVMLHeader{
		Magic:   AVMLMagic,
		Version: AVMLVersion,
		Start:   start,
		End:     inclusiveEnd,
	}
}

// ExclusiveEnd returns the exclusive end address.
func (h *AVMLHeader) ExclusiveEnd() uint64 {
	return h.End + 1
}

// RangeLen returns the length of the range.
func (h *AVMLHeader) RangeLen() uint64 {
	return h.ExclusiveEnd() - h.Start
}

// AVMLTrailerSize is the size of the compressed length trailer in bytes.
const AVMLTrailerSize = 8

// EncodeAVMLBlock writes an AVML-compressed block to w.
// It writes: header + 8-byte little-endian trailer + compressed payload.
// This ordering allows sequential reading without seeking.
func EncodeAVMLBlock(w io.Writer, header *AVMLHeader, data []byte) error {
	if err := header.Encode(w); err != nil {
		return fmt.Errorf("encode AVML header: %w", err)
	}

	// Compress data using Snappy.
	compressed := snappy.Encode(nil, data)

	// Write 8-byte little-endian trailer with compressed length.
	var trailer [AVMLTrailerSize]byte
	binary.LittleEndian.PutUint64(trailer[:], uint64(len(compressed)))
	if _, err := w.Write(trailer[:]); err != nil {
		return fmt.Errorf("write AVML trailer: %w", err)
	}

	// Write compressed payload.
	if _, err := w.Write(compressed); err != nil {
		return fmt.Errorf("write compressed payload: %w", err)
	}

	return nil
}

// DecodeAVMLBlock reads an AVML-compressed block from r.
// It reads: header + trailer + payload, then decompresses.
func DecodeAVMLBlock(r io.Reader) (*AVMLHeader, []byte, error) {
	header, err := DecodeAVMLHeader(r)
	if err != nil {
		return nil, nil, err
	}

	// Read trailer to get compressed length.
	var trailer [AVMLTrailerSize]byte
	if _, err := io.ReadFull(r, trailer[:]); err != nil {
		return nil, nil, fmt.Errorf("read AVML trailer: %w", err)
	}
	compressedLen := binary.LittleEndian.Uint64(trailer[:])

	// Read compressed payload.
	compressed := make([]byte, compressedLen)
	if _, err := io.ReadFull(r, compressed); err != nil {
		return nil, nil, fmt.Errorf("read compressed payload: %w", err)
	}

	// Decompress.
	data, err := snappy.Decode(nil, compressed)
	if err != nil {
		return nil, nil, fmt.Errorf("decompress snappy: %w", err)
	}

	return header, data, nil
}

// SplitRange splits a large range into AVML-compatible blocks.
func SplitRange(start, exclusiveEnd uint64) []struct{ Start, End uint64 } {
	var blocks []struct{ Start, End uint64 }
	for start < exclusiveEnd {
		end := start + AVMLBlockSize
		if end > exclusiveEnd {
			end = exclusiveEnd
		}
		blocks = append(blocks, struct{ Start, End uint64 }{start, end})
		start = end
	}
	return blocks
}
