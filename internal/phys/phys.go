// Package phys provides shared physical range types and utilities.
package phys

// Range represents a contiguous physical memory range with exclusive end.
type Range struct {
	Start uint64
	End   uint64 // exclusive
}

// Len returns the length of the range in bytes.
func (r Range) Len() uint64 {
	return r.End - r.Start
}

// IsEmpty returns true if the range has zero length.
func (r Range) IsEmpty() bool {
	return r.Start >= r.End
}

// Contains reports whether addr is within the range.
func (r Range) Contains(addr uint64) bool {
	return addr >= r.Start && addr < r.End
}

// Overlaps reports whether r and other share any addresses.
func (r Range) Overlaps(other Range) bool {
	return r.Start < other.End && other.Start < r.End
}

// Adjacent reports whether r and other are adjacent (r.End == other.Start or vice versa).
func (r Range) Adjacent(other Range) bool {
	return r.End == other.Start || other.End == r.Start
}

// Block represents a physical memory block with an associated range.
// The Data field is populated during acquisition and may be omitted
// for zero-block detection.
type Block struct {
	Range Range
	Data  []byte
}

// IsZero reports whether the block contains only zero bytes.
func (b Block) IsZero() bool {
	for _, v := range b.Data {
		if v != 0 {
			return false
		}
	}
	return true
}
