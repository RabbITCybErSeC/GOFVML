package procfs

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzParseMaps(f *testing.F) {
	// Seed corpus with valid and invalid maps lines.
	f.Add("00400000-00452000 r-xp 00000000 08:02 173521                           /usr/bin/dbus-daemon\n")
	f.Add("00651000-00652000 rw-p 00051000 08:02 173521                           /usr/bin/dbus-daemon\n")
	f.Add("7ffd6e5a8000-7ffd6e5c9000 rw-p 00000000 00:00 0                          [stack]\n")
	f.Add("malformed line\n")
	f.Add("")
	f.Add("00400000-00452000 r-xp 00000000 08:02 173521 /usr/bin/dbus-daemon extra\n")

	f.Fuzz(func(t *testing.T, data string) {
		// Write data to temp file for ParseMaps.
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "maps")
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		// ParseMaps should not panic on arbitrary input.
		_, _ = ParseMaps(f)
	})
}

func TestParseMaps_Malformed(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantLen int
	}{
		{
			name:    "too few fields",
			input:   "00400000-00452000 r-xp\n",
			wantErr: true,
			wantLen: 0,
		},
		{
			name:    "invalid address range",
			input:   "zzzz-00452000 r-xp 00000000 08:02 173521 /usr/bin/dbus-daemon\n",
			wantErr: true,
			wantLen: 0,
		},
		{
			name:    "missing hyphen in range",
			input:   "0040000000452000 r-xp 00000000 08:02 173521 /usr/bin/dbus-daemon\n",
			wantErr: true,
			wantLen: 0,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "valid single line",
			input:   "00400000-00452000 r-xp 00000000 08:02 173521 /usr/bin/dbus-daemon\n",
			wantErr: false,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "maps")
			if err := os.WriteFile(path, []byte(tt.input), 0600); err != nil {
				t.Fatal(err)
			}
			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			mappings, err := ParseMaps(f)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(mappings) != tt.wantLen {
				t.Errorf("expected %d mappings, got %d", tt.wantLen, len(mappings))
			}
		})
	}
}
