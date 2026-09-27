package escseq

import "io"

// maxPending caps how many bytes Writer will hold onto while waiting
// for an escape sequence to finish. Real sequences (even a long OSC
// hyperlink URL) end well under this; a stream that exceeds it without
// producing a terminator is either malformed or not actually escape
// data, so the held bytes are dropped instead of growing without
// bound.
const maxPending = 4096

// Writer wraps an underlying io.Writer, stripping ANSI/VT100 escape
// sequences from the bytes it's given before they reach it. It's for
// piping a program's output somewhere that shouldn't see raw color
// codes - a log file, a test recorder, a non-terminal destination.
//
// A single Write call is not the same unit as a single escape
// sequence: a colored line can arrive a chunk at a time, splitting a
// sequence across two or more calls. Writer buffers bytes from an
// unresolved ESC until either enough arrive to resolve it or the
// buffered amount exceeds maxPending, so nothing is forwarded (or
// dropped) before it's known whether those bytes are a sequence or
// plain text.
type Writer struct {
	w    io.Writer
	pend []byte // bytes from an unresolved ESC, held since the last Write
}

// NewWriter returns a Writer that strips escape sequences before
// passing the remaining bytes through to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Write implements io.Writer. Per the io.Writer contract it always
// reports having consumed the whole of p when err is nil, even though
// fewer bytes (or none) may reach the underlying writer: escape
// sequences are removed, and the tail of a sequence still in progress
// is held back rather than written.
func (wr *Writer) Write(p []byte) (int, error) {
	buf := p
	if len(wr.pend) > 0 {
		buf = append(wr.pend, p...)
		wr.pend = nil
	}

	var out []byte
	for i := 0; i < len(buf); {
		if buf[i] != 0x1b {
			out = append(out, buf[i])
			i++
			continue
		}

		n, complete := scanSeq(buf[i:])
		if !complete {
			if rest := buf[i:]; len(rest) <= maxPending {
				wr.pend = append(wr.pend, rest...)
			}
			break
		}
		i += n
	}

	if len(out) > 0 {
		if _, err := wr.w.Write(out); err != nil {
			return len(p), err
		}
	}
	return len(p), nil
}

// scanSeq is the streaming counterpart to seqLen: it reports both how
// many bytes the sequence starting at s[0] (which must be ESC) takes
// up, and whether a proper terminator was actually found. seqLen
// cannot answer the latter because Strip always operates on a
// complete, final string, where running out of bytes and being
// malformed amount to the same thing; a Writer needs to tell those
// apart so it knows whether to wait for more input.
func scanSeq(s []byte) (n int, complete bool) {
	if len(s) < 2 {
		return len(s), false
	}

	switch s[1] {
	case '[':
		return scanCSI(s)
	case ']', 'P', '^', '_':
		return scanString(s)
	default:
		return scanEsc(s)
	}
}

func scanCSI(s []byte) (int, bool) {
	i := 2
	for i < len(s) {
		if c := s[i]; c >= 0x40 && c <= 0x7e {
			return i + 1, true
		}
		i++
	}
	return i, false
}

func scanString(s []byte) (int, bool) {
	i := 2
	for i < len(s) {
		if s[i] == 0x07 {
			return i + 1, true
		}
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
			return i + 2, true
		}
		i++
	}
	return i, false
}

func scanEsc(s []byte) (int, bool) {
	i := 1
	for i < len(s) && s[i] >= 0x20 && s[i] <= 0x2f {
		i++
	}
	if i < len(s) {
		return i + 1, true
	}
	return i, false
}
