package image

import (
	"bytes"
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

var snappyStreamIdentifier = []byte{0xff, 0x06, 0x00, 0x00, 's', 'N', 'a', 'P', 'p', 'Y'}

// EncodeAVMLBlock writes an AVML-compressed block to w.
// It writes: header + Snappy framed payload + 8-byte little-endian trailer.
func EncodeAVMLBlock(w io.Writer, header *AVMLHeader, data []byte) error {
	if uint64(len(data)) != header.RangeLen() {
		return fmt.Errorf("AVML payload length %d does not match header range length %d", len(data), header.RangeLen())
	}

	if err := header.Encode(w); err != nil {
		return fmt.Errorf("encode AVML header: %w", err)
	}

	var compressed bytes.Buffer
	sw := snappy.NewBufferedWriter(&compressed)
	if _, err := sw.Write(data); err != nil {
		return fmt.Errorf("compress snappy payload: %w", err)
	}
	if err := sw.Close(); err != nil {
		return fmt.Errorf("finish snappy payload: %w", err)
	}

	if _, err := w.Write(compressed.Bytes()); err != nil {
		return fmt.Errorf("write compressed payload: %w", err)
	}

	var trailer [AVMLTrailerSize]byte
	binary.LittleEndian.PutUint64(trailer[:], uint64(compressed.Len()))
	if _, err := w.Write(trailer[:]); err != nil {
		return fmt.Errorf("write AVML trailer: %w", err)
	}

	return nil
}

// DecodeAVMLBlock reads an AVML-compressed block from r.
// It reads: header + Snappy framed payload + trailer, then decompresses.
func DecodeAVMLBlock(r io.Reader) (*AVMLHeader, []byte, error) {
	header, err := DecodeAVMLHeader(r)
	if err != nil {
		return nil, nil, err
	}

	compressed, data, err := readSnappyFrame(r, header.RangeLen())
	if err != nil {
		return nil, nil, err
	}

	var trailer [AVMLTrailerSize]byte
	if _, err := io.ReadFull(r, trailer[:]); err != nil {
		return nil, nil, fmt.Errorf("read AVML trailer: %w", err)
	}
	compressedLen := binary.LittleEndian.Uint64(trailer[:])
	if compressedLen != uint64(len(compressed)) {
		return nil, nil, fmt.Errorf("AVML trailer length %d does not match compressed payload length %d", compressedLen, len(compressed))
	}

	return header, data, nil
}

func readSnappyFrame(r io.Reader, wantLen uint64) ([]byte, []byte, error) {
	var compressed bytes.Buffer
	if _, err := io.CopyN(&compressed, r, int64(len(snappyStreamIdentifier))); err != nil {
		return nil, nil, fmt.Errorf("read snappy stream identifier: %w", err)
	}
	if !bytes.Equal(compressed.Bytes(), snappyStreamIdentifier) {
		return nil, nil, fmt.Errorf("invalid snappy stream identifier")
	}
	if wantLen == 0 {
		return compressed.Bytes(), nil, nil
	}

	var decoded []byte
	for {
		var chunkHeader [4]byte
		if _, err := io.ReadFull(r, chunkHeader[:]); err != nil {
			return nil, nil, fmt.Errorf("read snappy chunk header: %w", err)
		}
		compressed.Write(chunkHeader[:])

		chunkType := chunkHeader[0]
		chunkLen := int(chunkHeader[1]) | int(chunkHeader[2])<<8 | int(chunkHeader[3])<<16
		if chunkLen < 0 {
			return nil, nil, fmt.Errorf("invalid snappy chunk length %d", chunkLen)
		}
		chunk := make([]byte, chunkLen)
		if _, err := io.ReadFull(r, chunk); err != nil {
			return nil, nil, fmt.Errorf("read snappy chunk payload: %w", err)
		}
		compressed.Write(chunk)

		switch {
		case chunkType == 0x00 || chunkType == 0x01:
			var err error
			decoded, err = io.ReadAll(snappy.NewReader(bytes.NewReader(compressed.Bytes())))
			if err != nil {
				return nil, nil, fmt.Errorf("decompress snappy frame: %w", err)
			}
			if uint64(len(decoded)) == wantLen {
				return compressed.Bytes(), decoded, nil
			}
			if uint64(len(decoded)) > wantLen {
				return nil, nil, fmt.Errorf("decompressed payload length %d exceeds header range length %d", len(decoded), wantLen)
			}
		case chunkType == 0xff:
			return nil, nil, fmt.Errorf("unexpected snappy stream identifier inside block")
		case chunkType >= 0x80:
			// Skippable metadata chunks do not contribute decompressed bytes.
		default:
			return nil, nil, fmt.Errorf("unsupported snappy chunk type 0x%02x", chunkType)
		}
	}
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
