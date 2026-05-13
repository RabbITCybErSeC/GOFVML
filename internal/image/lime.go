// Package image provides LiME and AVML-compressed image encoders and decoders.
package image

import (
	"encoding/binary"
	"fmt"
	"io"
)

// LiMEHeaderSize is the size of a LiME header in bytes.
const LiMEHeaderSize = 32

// LiMEMagic is the LiME magic number ("LiME").
const LiMEMagic uint32 = 0x4c694d45

// LiMEVersion is the supported LiME version.
const LiMEVersion uint32 = 1

// LiMEHeader represents a LiME image block header.
type LiMEHeader struct {
	Magic   uint32
	Version uint32
	Start   uint64 // physical start address
	End     uint64 // physical end address (inclusive in LiME format)
}

// Encode writes the LiME header to w in little-endian format.
// The header is exactly 32 bytes.
func (h *LiMEHeader) Encode(w io.Writer) error {
	var buf [LiMEHeaderSize]byte
	binary.LittleEndian.PutUint32(buf[0:4], h.Magic)
	binary.LittleEndian.PutUint32(buf[4:8], h.Version)
	binary.LittleEndian.PutUint64(buf[8:16], h.Start)
	binary.LittleEndian.PutUint64(buf[16:24], h.End)
	// buf[24:32] is reserved (zero padding)
	_, err := w.Write(buf[:])
	return err
}

// DecodeLiMEHeader reads and decodes a LiME header from r.
func DecodeLiMEHeader(r io.Reader) (*LiMEHeader, error) {
	var buf [LiMEHeaderSize]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return nil, fmt.Errorf("read LiME header: %w", err)
	}

	h := &LiMEHeader{
		Magic:   binary.LittleEndian.Uint32(buf[0:4]),
		Version: binary.LittleEndian.Uint32(buf[4:8]),
		Start:   binary.LittleEndian.Uint64(buf[8:16]),
		End:     binary.LittleEndian.Uint64(buf[16:24]),
	}

	// Validate magic.
	if h.Magic != LiMEMagic {
		return nil, fmt.Errorf("invalid LiME magic: got 0x%08x, want 0x%08x", h.Magic, LiMEMagic)
	}

	// Validate version.
	if h.Version != LiMEVersion {
		return nil, fmt.Errorf("unsupported LiME version: got %d, want %d", h.Version, LiMEVersion)
	}

	// Validate reserved padding is zero.
	for i := 24; i < LiMEHeaderSize; i++ {
		if buf[i] != 0 {
			return nil, fmt.Errorf("non-zero padding at byte %d", i)
		}
	}

	// Validate range.
	if h.End < h.Start {
		return nil, fmt.Errorf("invalid range: end (0x%x) < start (0x%x)", h.End, h.Start)
	}

	return h, nil
}

// NewLiMEHeader creates a LiME header from a physical range.
// start and end use GOFVML's exclusive-end semantics and are converted
// to LiME's inclusive-end format.
func NewLiMEHeader(start, exclusiveEnd uint64) *LiMEHeader {
	// LiME stores inclusive end, so subtract 1 from exclusive end.
	// For empty ranges (start == exclusiveEnd), preserve the empty
	// range by keeping inclusiveEnd == start, which gives RangeLen() == 0.
	inclusiveEnd := start
	if exclusiveEnd > start {
		inclusiveEnd = exclusiveEnd - 1
	}

	return &LiMEHeader{
		Magic:   LiMEMagic,
		Version: LiMEVersion,
		Start:   start,
		End:     inclusiveEnd,
	}
}

// ExclusiveEnd returns the exclusive end address (GOFVML convention).
func (h *LiMEHeader) ExclusiveEnd() uint64 {
	return h.End + 1
}

// RangeLen returns the length of the range in bytes.
func (h *LiMEHeader) RangeLen() uint64 {
	return h.ExclusiveEnd() - h.Start
}

// IsEmpty reports whether the range has zero length.
func (h *LiMEHeader) IsEmpty() bool {
	return h.Start >= h.ExclusiveEnd()
}
