package process

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/RabbITCybErSeC/gofvml/internal/procfs"
)

// Process artifact format constants.
const (
	ArtifactMagic       = "gofvml-p1"
	ArtifactVersion     = 1
	ArtifactHeaderSize  = 32
	ArtifactFooterMagic = "GOFVML-EOF"
	ArtifactFooterSize  = 26 // 8 + 8 + 10
)

// CompressionType indicates the compression used for a payload block.
type CompressionType uint8

const (
	CompressionNone CompressionType = iota
)

// BlockStatus indicates the status of a payload block.
type BlockStatus uint8

const (
	StatusOK BlockStatus = iota
	StatusShortRead
	StatusError
	StatusUnmapped
)

// ArtifactHeader is the fixed-size file header.
type ArtifactHeader struct {
	Magic   [9]byte
	Version uint8
	_       [22]byte // reserved
}

// PayloadBlockHeader precedes each payload block.
type PayloadBlockHeader struct {
	VirtualAddress  uint64
	MappingIndex    uint32
	PayloadLength   uint32
	CompressionType uint8
	Status          uint8
	_               [2]byte // reserved, pad to 24 bytes
}

// PayloadBlock is a decoded payload block.
type PayloadBlock struct {
	VirtualAddress  uint64
	MappingIndex    uint32
	CompressionType CompressionType
	Status          BlockStatus
	Data            []byte
}

// ArtifactMetadata is the JSON metadata written at EOF.
type ArtifactMetadata struct {
	Version        string          `json:"version"`
	Host           string          `json:"host,omitempty"`
	Kernel         string          `json:"kernel,omitempty"`
	Timestamp      time.Time       `json:"timestamp"`
	PID            int             `json:"pid"`
	CommandLine    []string        `json:"command_line,omitempty"`
	ExecutablePath string          `json:"executable_path,omitempty"`
	Mappings       []procfs.Mapping `json:"mappings,omitempty"`
	ReadEvents     []ReadEvent     `json:"read_events,omitempty"`
	Strict         bool            `json:"strict"`
	BytesRead      uint64          `json:"bytes_read"`
}

// ArtifactFooter is the fixed-size footer at the very end of the file.
type ArtifactFooter struct {
	MetadataOffset uint64
	MetadataLength uint64
	Magic          [10]byte
}

// EncodeArtifactHeader writes the artifact header to w.
func EncodeArtifactHeader(w io.Writer) error {
	if _, err := w.Write([]byte(ArtifactMagic)); err != nil {
		return fmt.Errorf("write magic: %w", err)
	}
	if err := binary.Write(w, binary.LittleEndian, uint8(ArtifactVersion)); err != nil {
		return fmt.Errorf("write version: %w", err)
	}
	// Pad to ArtifactHeaderSize.
	padding := make([]byte, ArtifactHeaderSize-len(ArtifactMagic)-1)
	if _, err := w.Write(padding); err != nil {
		return fmt.Errorf("write header padding: %w", err)
	}
	return nil
}

// DecodeArtifactHeader reads the artifact header from r.
func DecodeArtifactHeader(r io.Reader) (*ArtifactHeader, error) {
	var h ArtifactHeader
	magicBytes := make([]byte, len(ArtifactMagic))
	if _, err := io.ReadFull(r, magicBytes); err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}
	copy(h.Magic[:], magicBytes)
	if string(h.Magic[:]) != ArtifactMagic {
		return nil, fmt.Errorf("invalid artifact magic: %q", string(h.Magic[:]))
	}
	var version uint8
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, fmt.Errorf("read version: %w", err)
	}
	h.Version = version
	if h.Version != ArtifactVersion {
		return nil, fmt.Errorf("unsupported artifact version: %d", h.Version)
	}
	// Skip padding.
	padding := make([]byte, ArtifactHeaderSize-len(ArtifactMagic)-1)
	if _, err := io.ReadFull(r, padding); err != nil {
		return nil, fmt.Errorf("read header padding: %w", err)
	}
	return &h, nil
}

// WritePayloadBlock writes a single payload block to w.
func WritePayloadBlock(w io.Writer, block PayloadBlock) error {
	if err := binary.Write(w, binary.LittleEndian, block.VirtualAddress); err != nil {
		return fmt.Errorf("write virtual address: %w", err)
	}
	if err := binary.Write(w, binary.LittleEndian, block.MappingIndex); err != nil {
		return fmt.Errorf("write mapping index: %w", err)
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(block.Data))); err != nil {
		return fmt.Errorf("write payload length: %w", err)
	}
	if err := binary.Write(w, binary.LittleEndian, uint8(block.CompressionType)); err != nil {
		return fmt.Errorf("write compression type: %w", err)
	}
	if err := binary.Write(w, binary.LittleEndian, uint8(block.Status)); err != nil {
		return fmt.Errorf("write status: %w", err)
	}
	// Pad to 24 bytes.
	if _, err := w.Write([]byte{0, 0}); err != nil {
		return fmt.Errorf("write header padding: %w", err)
	}
	if len(block.Data) > 0 {
		if _, err := w.Write(block.Data); err != nil {
			return fmt.Errorf("write payload data: %w", err)
		}
	}
	return nil
}

// ReadPayloadBlock reads a single payload block from r.
func ReadPayloadBlock(r io.Reader) (*PayloadBlock, error) {
	var block PayloadBlock
	if err := binary.Read(r, binary.LittleEndian, &block.VirtualAddress); err != nil {
		return nil, fmt.Errorf("read virtual address: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &block.MappingIndex); err != nil {
		return nil, fmt.Errorf("read mapping index: %w", err)
	}
	var payloadLength uint32
	if err := binary.Read(r, binary.LittleEndian, &payloadLength); err != nil {
		return nil, fmt.Errorf("read payload length: %w", err)
	}
	var compressionType uint8
	if err := binary.Read(r, binary.LittleEndian, &compressionType); err != nil {
		return nil, fmt.Errorf("read compression type: %w", err)
	}
	block.CompressionType = CompressionType(compressionType)
	var status uint8
	if err := binary.Read(r, binary.LittleEndian, &status); err != nil {
		return nil, fmt.Errorf("read status: %w", err)
	}
	block.Status = BlockStatus(status)
	// Skip padding.
	padding := make([]byte, 2)
	if _, err := io.ReadFull(r, padding); err != nil {
		return nil, fmt.Errorf("read header padding: %w", err)
	}
	if payloadLength > 0 {
		block.Data = make([]byte, payloadLength)
		if _, err := io.ReadFull(r, block.Data); err != nil {
			return nil, fmt.Errorf("read payload data: %w", err)
		}
	}
	return &block, nil
}

// WriteArtifact writes a complete artifact to w.
// It writes the header, payload blocks, metadata, and footer.
func WriteArtifact(w io.Writer, blocks []PayloadBlock, meta ArtifactMetadata) error {
	if err := EncodeArtifactHeader(w); err != nil {
		return err
	}

	for i, block := range blocks {
		if err := WritePayloadBlock(w, block); err != nil {
			return fmt.Errorf("write block %d: %w", i, err)
		}
	}

	meta.Version = fmt.Sprintf("gofvml-process-v%d", ArtifactVersion)
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	// Write metadata.
	if _, err := w.Write(metaJSON); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	// Write footer.
	metadataOffset := uint64(ArtifactHeaderSize)
	for _, block := range blocks {
		metadataOffset += uint64(binary.Size(PayloadBlockHeader{}) + len(block.Data))
	}
	if err := binary.Write(w, binary.LittleEndian, metadataOffset); err != nil {
		return fmt.Errorf("write footer offset: %w", err)
	}
	if err := binary.Write(w, binary.LittleEndian, uint64(len(metaJSON))); err != nil {
		return fmt.Errorf("write footer length: %w", err)
	}
	if _, err := w.Write([]byte(ArtifactFooterMagic)); err != nil {
		return fmt.Errorf("write footer magic: %w", err)
	}

	return nil
}

// ReadArtifact reads a complete artifact from a seekable reader.
func ReadArtifact(r io.ReadSeeker) (*ArtifactMetadata, []PayloadBlock, error) {
	// Read and verify header.
	if _, err := DecodeArtifactHeader(r); err != nil {
		return nil, nil, err
	}

	// Seek to footer.
	if _, err := r.Seek(-int64(ArtifactFooterSize), io.SeekEnd); err != nil {
		return nil, nil, fmt.Errorf("seek to footer: %w", err)
	}
	var footer ArtifactFooter
	if err := binary.Read(r, binary.LittleEndian, &footer.MetadataOffset); err != nil {
		return nil, nil, fmt.Errorf("read footer offset: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &footer.MetadataLength); err != nil {
		return nil, nil, fmt.Errorf("read footer length: %w", err)
	}
	magicBytes := make([]byte, 10)
	if _, err := io.ReadFull(r, magicBytes); err != nil {
		return nil, nil, fmt.Errorf("read footer magic: %w", err)
	}
	copy(footer.Magic[:], magicBytes)
	if string(footer.Magic[:]) != ArtifactFooterMagic {
		return nil, nil, fmt.Errorf("invalid footer magic: %q", string(footer.Magic[:]))
	}

	// Read metadata.
	if _, err := r.Seek(int64(footer.MetadataOffset), io.SeekStart); err != nil {
		return nil, nil, fmt.Errorf("seek to metadata: %w", err)
	}
	metaBytes := make([]byte, footer.MetadataLength)
	if _, err := io.ReadFull(r, metaBytes); err != nil {
		return nil, nil, fmt.Errorf("read metadata: %w", err)
	}
	var meta ArtifactMetadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	// Read payload blocks.
	if _, err := r.Seek(int64(ArtifactHeaderSize), io.SeekStart); err != nil {
		return nil, nil, fmt.Errorf("seek to blocks: %w", err)
	}

	var blocks []PayloadBlock
	for {
		currentOffset, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, nil, err
		}
		if uint64(currentOffset) >= footer.MetadataOffset {
			break
		}
		block, err := ReadPayloadBlock(r)
		if err != nil {
			return nil, nil, fmt.Errorf("read block at offset %d: %w", currentOffset, err)
		}
		blocks = append(blocks, *block)
	}

	return &meta, blocks, nil
}
