// Package procfs provides focused /proc filesystem readers.
package procfs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Status holds selected fields from /proc/<pid>/status.
type Status struct {
	Name  string
	State string
	PPid  int
}

// ReadStatus reads /proc/<pid>/status and returns selected fields.
func ReadStatus(pid int) (*Status, error) {
	path := filepath.Join("/proc", strconv.Itoa(pid), "status")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var s Status
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Name:") {
			s.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		} else if strings.HasPrefix(line, "State:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				s.State = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "PPid:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				v, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					s.PPid = v
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return &s, nil
}

// ReadCmdline reads /proc/<pid>/cmdline and returns the argument slice.
// Arguments are separated by null bytes.
func ReadCmdline(pid int) ([]string, error) {
	path := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return nil, nil
	}
	// Remove trailing null byte(s) to avoid empty last element.
	for len(data) > 0 && data[len(data)-1] == 0 {
		data = data[:len(data)-1]
	}
	args := strings.Split(string(data), "\x00")
	return args, nil
}

// ReadExe returns the target of /proc/<pid>/exe symlink.
func ReadExe(pid int) (string, error) {
	path := filepath.Join("/proc", strconv.Itoa(pid), "exe")
	target, err := os.Readlink(path)
	if err != nil {
		return "", fmt.Errorf("readlink %s: %w", path, err)
	}
	return target, nil
}

// Mapping represents a single memory mapping from /proc/<pid>/maps.
type Mapping struct {
	Start    uint64
	End      uint64
	Perms    string
	Offset   uint64
	DevMajor int
	DevMinor int
	Inode    uint64
	Pathname string
}

// IsReadable reports whether the mapping has read permission.
func (m Mapping) IsReadable() bool {
	return strings.ContainsRune(m.Perms, 'r')
}

// IsWritable reports whether the mapping has write permission.
func (m Mapping) IsWritable() bool {
	return strings.ContainsRune(m.Perms, 'w')
}

// IsExecutable reports whether the mapping has execute permission.
func (m Mapping) IsExecutable() bool {
	return strings.ContainsRune(m.Perms, 'x')
}

// IsPrivate reports whether the mapping is private (copy-on-write).
func (m Mapping) IsPrivate() bool {
	return strings.ContainsRune(m.Perms, 'p')
}

// IsShared reports whether the mapping is shared.
func (m Mapping) IsShared() bool {
	return strings.ContainsRune(m.Perms, 's')
}

// Len returns the length of the mapping in bytes.
func (m Mapping) Len() uint64 {
	return m.End - m.Start
}

// ReadMaps reads /proc/<pid>/maps and returns all mappings.
func ReadMaps(pid int) ([]Mapping, error) {
	path := filepath.Join("/proc", strconv.Itoa(pid), "maps")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return ParseMaps(f)
}

// ParseMaps parses memory mappings from an io.Reader.
// The reader should provide /proc/<pid>/maps formatted lines.
func ParseMaps(r *os.File) ([]Mapping, error) {
	var mappings []Mapping
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		m, err := parseMapLine(line)
		if err != nil {
			return nil, fmt.Errorf("parse maps line %q: %w", line, err)
		}
		mappings = append(mappings, m)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read maps: %w", err)
	}
	return mappings, nil
}

// parseMapLine parses a single /proc/<pid>/maps line.
func parseMapLine(line string) (Mapping, error) {
	var m Mapping
	// Expected format:
	// start-end perms offset devMajor:devMinor inode pathname
	// e.g. 55f3c1a2b000-55f3c1a2c000 r--p 00000000 08:01 1310734 /usr/bin/cat
	// pathname may be absent.

	parts := strings.Fields(line)
	if len(parts) < 5 {
		return m, fmt.Errorf("expected at least 5 fields, got %d", len(parts))
	}

	// Parse address range: start-end
	addrParts := strings.Split(parts[0], "-")
	if len(addrParts) != 2 {
		return m, fmt.Errorf("invalid address range %q", parts[0])
	}
	start, err := strconv.ParseUint(addrParts[0], 16, 64)
	if err != nil {
		return m, fmt.Errorf("invalid start address %q: %w", addrParts[0], err)
	}
	end, err := strconv.ParseUint(addrParts[1], 16, 64)
	if err != nil {
		return m, fmt.Errorf("invalid end address %q: %w", addrParts[1], err)
	}
	if end <= start {
		return m, fmt.Errorf("invalid range: end %x <= start %x", end, start)
	}
	m.Start = start
	m.End = end

	// Permissions
	m.Perms = parts[1]

	// Offset
	offset, err := strconv.ParseUint(parts[2], 16, 64)
	if err != nil {
		return m, fmt.Errorf("invalid offset %q: %w", parts[2], err)
	}
	m.Offset = offset

	// Device major:minor
	devParts := strings.Split(parts[3], ":")
	if len(devParts) != 2 {
		return m, fmt.Errorf("invalid device %q", parts[3])
	}
	major, err := strconv.ParseInt(devParts[0], 16, 32)
	if err != nil {
		return m, fmt.Errorf("invalid device major %q: %w", devParts[0], err)
	}
	minor, err := strconv.ParseInt(devParts[1], 16, 32)
	if err != nil {
		return m, fmt.Errorf("invalid device minor %q: %w", devParts[1], err)
	}
	m.DevMajor = int(major)
	m.DevMinor = int(minor)

	// Inode
	inode, err := strconv.ParseUint(parts[4], 10, 64)
	if err != nil {
		return m, fmt.Errorf("invalid inode %q: %w", parts[4], err)
	}
	m.Inode = inode

	// Pathname (remainder of line, may be absent)
	if len(parts) > 5 {
		// Find the end of the 5th field in the original line.
		// Fields are: addr, perms, offset, dev, inode, [pathname...]
		pos := 0
		fieldsSeen := 0
		for pos < len(line) && fieldsSeen < 5 {
			// Skip whitespace
			for pos < len(line) && (line[pos] == ' ' || line[pos] == '\t') {
				pos++
			}
			if pos >= len(line) {
				break
			}
			// Skip non-whitespace (field content)
			for pos < len(line) && line[pos] != ' ' && line[pos] != '\t' {
				pos++
			}
			fieldsSeen++
			if fieldsSeen == 5 {
				m.Pathname = strings.TrimSpace(line[pos:])
			}
		}
	}

	return m, nil
}

// ReadYamaPtraceScope reads /proc/sys/kernel/yama/ptrace_scope.
// Returns the integer value (0-3 typically) or an error.
func ReadYamaPtraceScope() (int, error) {
	const path = "/proc/sys/kernel/yama/ptrace_scope"
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	v, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", path, err)
	}
	return v, nil
}
