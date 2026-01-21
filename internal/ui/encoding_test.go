package ui

import "testing"

func TestNormalizeEncoding(t *testing.T) {
	cases := map[string]string{
		"utf-8": "UTF-8",
		"UTF8":  "UTF8",
		" iso_8859-1 ": "ISO-8859-1",
	}
	for input, want := range cases {
		if got := normalizeEncoding(input); got != want {
			t.Fatalf("normalizeEncoding(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestConvertToUTF8ISO88591(t *testing.T) {
	input := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0xff} // "Hello" + 0xFF
	converted := convertToUTF8("ISO-8859-1", input)
	if len(converted) <= len(input) {
		t.Fatalf("convertToUTF8 should expand non-ASCII bytes")
	}
}

