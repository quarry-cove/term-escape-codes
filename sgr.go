package escseq

import (
	"strconv"
	"strings"
)

// SGR (Select Graphic Rendition) codes. This is the subset that covers
// ordinary text styling and the 16-color palette; see ECMA-48 section
// 8.3.117 for the full list.
const (
	CodeReset     = 0
	CodeBold      = 1
	CodeFaint     = 2
	CodeItalic    = 3
	CodeUnderline = 4
	CodeBlink     = 5
	CodeReverse   = 7
	CodeStrike    = 9

	CodeFgBlack   = 30
	CodeFgRed     = 31
	CodeFgGreen   = 32
	CodeFgYellow  = 33
	CodeFgBlue    = 34
	CodeFgMagenta = 35
	CodeFgCyan    = 36
	CodeFgWhite   = 37
	CodeFgDefault = 39

	CodeBgBlack   = 40
	CodeBgRed     = 41
	CodeBgGreen   = 42
	CodeBgYellow  = 43
	CodeBgBlue    = 44
	CodeBgMagenta = 45
	CodeBgCyan    = 46
	CodeBgWhite   = 47
	CodeBgDefault = 49
)

// SGR builds a single CSI escape sequence from one or more SGR codes,
// e.g. SGR(CodeBold, CodeFgRed) produces "\x1b[1;31m". Calling it with
// no codes produces a full reset, since an omitted parameter list means
// exactly that in the spec.
func SGR(codes ...int) string {
	if len(codes) == 0 {
		codes = []int{CodeReset}
	}
	parts := make([]string, len(codes))
	for i, c := range codes {
		parts[i] = strconv.Itoa(c)
	}
	return ESC + "[" + strings.Join(parts, ";") + "m"
}

// Fg256 and Bg256 select from the 256-color palette (0-255): 0-15 are
// the standard/bright ANSI colors, 16-231 a 6x6x6 color cube, and
// 232-255 a grayscale ramp.
func Fg256(n uint8) string { return ESC + "[38;5;" + strconv.Itoa(int(n)) + "m" }
func Bg256(n uint8) string { return ESC + "[48;5;" + strconv.Itoa(int(n)) + "m" }

// FgRGB and BgRGB build 24-bit truecolor sequences using the widely
// supported (if never formally standardized by ECMA-48) 38;2;r;g;b
// form.
func FgRGB(r, g, b uint8) string { return ESC + "[38;2;" + rgb(r, g, b) + "m" }
func BgRGB(r, g, b uint8) string { return ESC + "[48;2;" + rgb(r, g, b) + "m" }

func rgb(r, g, b uint8) string {
	return strconv.Itoa(int(r)) + ";" + strconv.Itoa(int(g)) + ";" + strconv.Itoa(int(b))
}
