// Package conversion provides image format conversion between raw, LiME, and AVML.
package conversion

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Format represents a memory image format.
type Format int

const (
	// FormatUnknown indicates an unrecognized format.
	FormatUnknown Format = iota
	// FormatRaw is a raw physical memory image.
	FormatRaw
	// FormatLiME is a LiME-encoded image.
	FormatLiME
	// FormatAVML is an AVML-compressed image.
	FormatAVML
)

func (f Format) String() string {
	switch f {
	case FormatRaw:
		return "raw"
	case FormatLiME:
		return "lime"
	case FormatAVML:
		return "avml"
	default:
		return "unknown"
	}
}

// DetectFormat reads the beginning of r to determine the image format.
// It returns the format and any error encountered.
func DetectFormat(r io.Reader) (Format, error) {
	// Read first 4 bytes to check magic.
	var magic [4]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		if errors.Is(err, io.EOF) {
			return FormatUnknown, fmt.Errorf("empty input")
		}
		return FormatUnknown, fmt.Errorf("read magic: %w", err)
	}

	switch binary.LittleEndian.Uint32(magic[:]) {
	case 0x4c694d45: // "LiME"
		return FormatLiME, nil
	case 0x41564d4c: // "AVML"
		return FormatAVML, nil
	default:
		// Could be raw (no magic) or unknown.
		return FormatRaw, nil
	}
}

// DetectFormatFromBytes detects format from a byte slice.
func DetectFormatFromBytes(data []byte) (Format, error) {
	return DetectFormat(bytes.NewReader(data))
}

// ValidateFormatPair checks if source and target formats are compatible
// for conversion. Returns an error if they are the same (no-op) or if
// either is unknown.
func ValidateFormatPair(source, target Format) error {
	if source == FormatUnknown {
		return fmt.Errorf("unknown source format")
	}
	if target == FormatUnknown {
		return fmt.Errorf("unknown target format")
	}
	if source == target {
		return fmt.Errorf("source and target format are both %s: no conversion needed", source)
	}
	return nil
}
