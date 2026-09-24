// Package escseq builds and strips ANSI/VT100 terminal escape sequences:
// the codes a program writes to color text, move the cursor, or set a
// window title, mixed in with otherwise ordinary output.
//
// The recurring problem this solves: once a string has escape codes in
// it, you can no longer trust len(s) or column-align it, because bytes
// that are never drawn on screen are sitting there taking up space.
// Strip and VisibleLen exist to answer "what does this actually look
// like on screen" for logging, testing CLI output, or laying out a
// table that contains colored cells.
package escseq

import (
	"strings"
	"unicode/utf8"
)

// ESC is the byte that introduces every sequence this package deals
// with. All the constructors and the parser below key off it.
const ESC = "\x1b"

// Strip removes escape sequences from s, returning the text a human
// would actually see if s were printed to a terminal. Plain text is
// returned unchanged (and unallocated).
//
// This only recognizes 7-bit sequences, i.e. ones that start with the
// two bytes ESC (0x1b) plus a following byte. It deliberately does not
// treat the 8-bit C1 control codes (0x80-0x9f, which include a
// single-byte form of CSI and OSC) as escape introducers: in UTF-8
// text those byte values only ever show up as continuation bytes
// inside a multi-byte rune, and stripping on sight of one would
// silently corrupt real characters. Terminals and terminal-writing
// programs in the wild overwhelmingly use the 7-bit forms for exactly
// this reason.
func Strip(s string) string {
	if !strings.ContainsRune(s, 0x1b) {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			i += seqLen(s[i:])
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// VisibleLen returns the number of runes left after Strip. It is a
// stand-in for "how much horizontal space does this take up", not an
// exact one: wide runes (CJK, emoji) and combining marks aren't
// accounted for, so a fixed-width table built from this will still be
// off for that content.
func VisibleLen(s string) int {
	return utf8.RuneCountInString(Strip(s))
}

// seqLen returns the number of bytes consumed by the escape sequence
// starting at s[0], which must be ESC. Malformed or truncated input
// (a sequence cut off mid-stream, which happens whenever a long write
// gets split across a buffer boundary) is consumed rather than left to
// confuse the next iteration or panic on an out-of-range index.
func seqLen(s string) int {
	if len(s) < 2 {
		return len(s) // lone ESC at end of input
	}

	switch s[1] {
	case '[':
		return csiLen(s)
	case ']', 'P', '^', '_':
		return stringLen(s)
	default:
		return escLen(s)
	}
}

// csiLen handles CSI sequences: ESC [ followed by any number of
// parameter/intermediate bytes and a single final byte in 0x40-0x7e,
// e.g. "\x1b[38;5;196m" or the private-mode "\x1b[?25l".
func csiLen(s string) int {
	i := 2 // past ESC [
	for i < len(s) {
		if c := s[i]; c >= 0x40 && c <= 0x7e {
			return i + 1
		}
		i++
	}
	return i // truncated: no final byte, consume what there is
}

// stringLen handles the "string" sequences: OSC (window title, hyperlinks),
// DCS, PM and APC. All of them run until a full ST (ESC \); OSC also
// accepts a bare BEL as a widely used xterm shorthand for the same thing.
func stringLen(s string) int {
	i := 2
	for i < len(s) {
		if s[i] == 0x07 {
			return i + 1
		}
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
			return i + 2
		}
		i++
	}
	return i // truncated: no terminator found, consume the rest
}

// escLen handles the short two-byte sequences (cursor save/restore,
// full reset, keypad mode, ...) as well as "nF" sequences such as
// "\x1b(B" (select ASCII as G0) that carry one or more intermediate
// bytes (0x20-0x2f) before their final byte.
func escLen(s string) int {
	i := 1
	for i < len(s) && s[i] >= 0x20 && s[i] <= 0x2f {
		i++
	}
	if i < len(s) {
		i++ // consume the final byte
	}
	return i
}
