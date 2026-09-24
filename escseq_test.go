package escseq

import "testing"

func TestStrip(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain text, no escapes",
			in:   "hello world",
			want: "hello world",
		},
		{
			name: "single SGR sequence",
			in:   "\x1b[31mred\x1b[0m",
			want: "red",
		},
		{
			name: "back-to-back sequences with no text between them",
			in:   "\x1b[1m\x1b[31mbold red\x1b[0m\x1b[0m",
			want: "bold red",
		},
		{
			name: "CSI with a private-mode marker (cursor hide)",
			in:   "before\x1b[?25lafter",
			want: "beforeafter",
		},
		{
			name: "CSI with multiple numeric params (256-color)",
			in:   "\x1b[38;5;196mred256\x1b[0m",
			want: "red256",
		},
		{
			name: "CSI truecolor params",
			in:   "\x1b[38;2;255;0;0mtruecolor\x1b[0m",
			want: "truecolor",
		},
		{
			name: "OSC hyperlink terminated by BEL",
			in:   "\x1b]8;;http://example.com\x07link text\x1b]8;;\x07",
			want: "link text",
		},
		{
			name: "OSC window title terminated by ST",
			in:   "\x1b]0;my title\x1b\\rest of line",
			want: "rest of line",
		},
		{
			name: "two-byte sequence (cursor save/restore)",
			in:   "\x1b7moved\x1b8",
			want: "moved",
		},
		{
			name: "nF sequence selecting a character set",
			in:   "\x1b(Bascii\x1b)0",
			want: "ascii",
		},
		{
			name: "truncated CSI sequence with no final byte",
			in:   "keep\x1b[3",
			want: "keep",
		},
		{
			name: "lone ESC at the very end of the string",
			in:   "keep\x1b",
			want: "keep",
		},
		{
			name: "truncated OSC sequence with no terminator",
			in:   "keep\x1b]0;untitled",
			want: "keep",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
		// Greek capital Lambda (U+039B) encodes in UTF-8 as the two
		// bytes 0xCE 0x9B. 0x9B is also the C1 (8-bit) form of CSI.
		// This package only recognizes the 7-bit ESC-prefixed form, so
		// this rune must survive untouched rather than have its second
		// byte mistaken for an escape introducer.
		{
			name: "multi-byte rune whose second byte collides with a C1 code",
			in:   "\x1b[31mΛ\x1b[0m",
			want: "Λ",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Strip(tc.in)
			if got != tc.want {
				t.Errorf("Strip(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestVisibleLen(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"plain ascii", "hello", 5},
		{"colored ascii", "\x1b[31mhello\x1b[0m", 5},
		{"multi-byte rune survives counting", "\x1b[31mΛ\x1b[0m", 1},
		{"empty", "", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := VisibleLen(tc.in); got != tc.want {
				t.Errorf("VisibleLen(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestSGR(t *testing.T) {
	cases := []struct {
		name  string
		codes []int
		want  string
	}{
		{"no codes defaults to reset", nil, "\x1b[0m"},
		{"single code", []int{CodeBold}, "\x1b[1m"},
		{"multiple codes joined with semicolons", []int{CodeBold, CodeFgRed}, "\x1b[1;31m"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SGR(tc.codes...); got != tc.want {
				t.Errorf("SGR(%v) = %q, want %q", tc.codes, got, tc.want)
			}
		})
	}
}

func TestRoundTripThroughStrip(t *testing.T) {
	// Anything this package builds should disappear completely under
	// Strip, leaving only the plain text it was wrapped around.
	built := SGR(CodeBold, CodeFgRed) + "warning" + SGR()
	if got, want := Strip(built), "warning"; got != want {
		t.Errorf("Strip(%q) = %q, want %q", built, got, want)
	}

	built = Fg256(196) + "x" + SGR() + Bg256(21) + "y" + SGR()
	if got, want := Strip(built), "xy"; got != want {
		t.Errorf("Strip(%q) = %q, want %q", built, got, want)
	}

	built = FgRGB(255, 0, 0) + "z" + SGR()
	if got, want := Strip(built), "z"; got != want {
		t.Errorf("Strip(%q) = %q, want %q", built, got, want)
	}
}
