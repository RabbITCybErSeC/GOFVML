package image

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestLiMEHeader_Malformed(t *testing.T) {
	tests := []struct {
		name    string
		modify  func([]byte)
		wantErr string
	}{
		{
			name: "wrong magic",
			modify: func(b []byte) {
				b[0] = 0xDE
				b[1] = 0xAD
				b[2] = 0xBE
				b[3] = 0xEF
			},
			wantErr: "invalid LiME magic",
		},
		{
			name: "wrong version",
			modify: func(b []byte) {
				b[4] = 0x99
			},
			wantErr: "unsupported LiME version",
		},
		{
			name: "non-zero padding",
			modify: func(b []byte) {
				b[24] = 0x01
			},
			wantErr: "non-zero padding",
		},
		{
			name: "reversed range",
			modify: func(b []byte) {
				binary.LittleEndian.PutUint64(b[8:16], 0x2000)
				binary.LittleEndian.PutUint64(b[16:24], 0x1000)
			},
			wantErr: "invalid range",
		},
		{
			name: "truncated header",
			modify: func(b []byte) {
				// Return a short slice in the reader.
			},
			wantErr: "read LiME header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			h := NewLiMEHeader(0x1000, 0x2000)
			if err := h.Encode(&buf); err != nil {
				t.Fatal(err)
			}

			data := buf.Bytes()
			if tt.name != "truncated header" {
				tt.modify(data)
			} else {
				data = data[:20] // truncated
			}

			_, err := DecodeLiMEHeader(bytes.NewReader(data))
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantErr != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.wantErr)) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestAVMLHeader_Malformed(t *testing.T) {
	tests := []struct {
		name    string
		modify  func([]byte)
		wantErr string
	}{
		{
			name: "wrong magic",
			modify: func(b []byte) {
				b[0] = 0xDE
				b[1] = 0xAD
				b[2] = 0xBE
				b[3] = 0xEF
			},
			wantErr: "invalid AVML magic",
		},
		{
			name: "wrong version",
			modify: func(b []byte) {
				b[4] = 0x99
			},
			wantErr: "unsupported AVML version",
		},
		{
			name: "non-zero padding",
			modify: func(b []byte) {
				b[24] = 0x01
			},
			wantErr: "non-zero padding",
		},
		{
			name: "reversed range",
			modify: func(b []byte) {
				binary.LittleEndian.PutUint64(b[8:16], 0x2000)
				binary.LittleEndian.PutUint64(b[16:24], 0x1000)
			},
			wantErr: "invalid range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			h := NewAVMLHeader(0x1000, 0x2000)
			if err := h.Encode(&buf); err != nil {
				t.Fatal(err)
			}

			data := buf.Bytes()
			tt.modify(data)

			_, err := DecodeAVMLHeader(bytes.NewReader(data))
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantErr != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.wantErr)) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestDecodeAVMLBlock_Malformed(t *testing.T) {
	// Valid header but truncated payload/trailer.
	var buf bytes.Buffer
	h := NewAVMLHeader(0x1000, 0x2000)
	if err := h.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	// Write partial snappy stream identifier.
	buf.Write([]byte{0xff, 0x06, 0x00})

	_, _, err := DecodeAVMLBlock(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("expected error for truncated payload")
	}
}
