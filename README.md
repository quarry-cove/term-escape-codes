# term-escape-codes

A small Go library for building and stripping terminal escape sequences
(the codes that color text, move the cursor, or set a window title).

## the problem

Once a string has escape codes mixed into it, `len(s)` stops meaning
anything useful. A colored log line, a progress bar, or a table cell
built with ANSI colors will report a length that includes bytes nobody
ever sees on screen. Column alignment breaks, diffing breaks, and tests
that assert on captured CLI output become unreadable because they have
to embed raw `\x1b[...m` bytes.

This package is the part of a CLI tool you'd otherwise end up writing
from scratch: a way to build the sequences you want to emit, and a way
to strip or measure them back out again for logging and testing.

## usage

Building sequences:

```go
package main

import (
	"fmt"

	"github.com/quarry-cove/term-escape-codes"
)

func main() {
	fmt.Println(escseq.SGR(escseq.CodeBold, escseq.CodeFgRed) + "warning" + escseq.SGR())
	fmt.Println(escseq.Fg256(208) + "orange, 256-color palette" + escseq.SGR())
	fmt.Println(escseq.FgRGB(64, 200, 255) + "truecolor" + escseq.SGR())
}
```

Stripping them back out, e.g. to compare captured CLI output against a
plain-text expectation in a test:

```go
got := runCLI() // returns something like "\x1b[32mOK\x1b[0m: 3 tests passed"
if escseq.Strip(got) != "OK: 3 tests passed" {
	t.Errorf("unexpected output: %q", got)
}
```

Measuring how much space a string actually takes up on screen:

```go
label := escseq.SGR(escseq.CodeBold) + "Status" + escseq.SGR()
pad := 12 - escseq.VisibleLen(label)
fmt.Println(label + strings.Repeat(" ", pad) + "OK")
```

## what Strip does and doesn't handle

`Strip` recognizes CSI sequences (`ESC [ ... final-byte`, used for
color and cursor movement), OSC/DCS/PM/APC "string" sequences (`ESC ]`
and friends, terminated by BEL or `ESC \`, used for window titles and
hyperlinks), and the short two-byte and `nF` sequences used for things
like cursor save/restore and character set selection.

It only recognizes the 7-bit, ESC-prefixed forms of these. Some
terminals also support 8-bit "C1" control codes, where a single byte
in the 0x80-0x9f range stands in for `ESC` plus a following character.
This package does not treat those as escape introducers: in UTF-8 text
(which is what this package assumes throughout) those same byte values
only ever appear as continuation bytes inside an ordinary multi-byte
rune, and stripping on sight of one would corrupt real text rather
than remove an escape code. See the test case built around the Greek
letter Lambda (U+039B, which happens to UTF-8-encode to a byte
matching the C1 form of CSI) for a concrete example of why this
matters.

`VisibleLen` counts runes after stripping. It is a reasonable proxy for
"how wide is this on screen" but isn't exact: wide characters (CJK,
many emoji) and combining marks aren't accounted for yet.

## license

MIT, see LICENSE.
