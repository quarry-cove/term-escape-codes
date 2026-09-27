package escseq

import (
	"bytes"
	"testing"
)

func TestWriter(t *testing.T) {
	// A single write, or several writes concatenated, whose combined
	// bytes should produce want once escapes are stripped.
	cases := []struct {
		name   string
		writes []string
		want   string
	}{
		{
			name:   "plain text",
			writes: []string{"hello world"},
			want:   "hello world",
		},
		{
			name:   "complete sequence in one write",
			writes: []string{"\x1b[31mred\x1b[0m"},
			want:   "red",
		},
		{
			name:   "CSI sequence split across two writes",
			writes: []string{"before\x1b[3", "1mafter\x1b[0m"},
			want:   "beforeafter",
		},
		{
			name:   "CSI sequence split byte by byte",
			writes: []string{"\x1b", "[", "3", "1", "m", "red"},
			want:   "red",
		},
		{
			name:   "OSC sequence split before its BEL terminator",
			writes: []string{"\x1b]8;;http://example.com", "\x07link\x1b]8;;\x07"},
			want:   "link",
		},
		{
			name:   "OSC sequence split before its ST terminator",
			writes: []string{"\x1b]0;title", "\x1b\\rest"},
			want:   "rest",
		},
		{
			name:   "text and sequences interleaved across many writes",
			writes: []string{"a", "\x1b[1m", "b", "\x1b[0m", "c"},
			want:   "abc",
		},
		{
			name:   "lone ESC held back and never resolved",
			writes: []string{"keep\x1b"},
			want:   "keep",
		},
		{
			name:   "empty write",
			writes: []string{""},
			want:   "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := NewWriter(&buf)
			for _, chunk := range tc.writes {
				n, err := w.Write([]byte(chunk))
				if err != nil {
					t.Fatalf("Write(%q) returned error: %v", chunk, err)
				}
				if n != len(chunk) {
					t.Errorf("Write(%q) = %d, want %d (must report the full length consumed)", chunk, n, len(chunk))
				}
			}
			if got := buf.String(); got != tc.want {
				t.Errorf("output = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWriter_UnresolvedSequenceBeyondCapIsDropped(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	// A CSI sequence with far more parameter bytes than any real
	// terminal escape carries, and no final byte: it should be
	// dropped once it exceeds maxPending rather than buffered
	// forever or leaked through as plain text.
	long := "\x1b[" + string(bytes.Repeat([]byte("9"), maxPending+1))
	if _, err := w.Write([]byte(long)); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if _, err := w.Write([]byte("done")); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if got, want := buf.String(), "done"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}
