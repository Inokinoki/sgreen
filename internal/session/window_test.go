package session

import "testing"

func TestWindowNumberConversion(t *testing.T) {
	cases := []struct {
		id  int
		str string
	}{
		{0, "0"},
		{9, "9"},
		{10, "A"},
		{11, "B"},
		{35, "Z"},
	}

	for _, c := range cases {
		got := windowNumberToString(c.id)
		if got != c.str {
			t.Fatalf("windowNumberToString(%d) = %s, want %s", c.id, got, c.str)
		}
		back, err := windowStringToNumber(c.str)
		if err != nil {
			t.Fatalf("windowStringToNumber(%s) error: %v", c.str, err)
		}
		if back != c.id {
			t.Fatalf("windowStringToNumber(%s) = %d, want %d", c.str, back, c.id)
		}
	}
}

func TestDetectEncodingFromLocale(t *testing.T) {
	t.Setenv("LC_ALL", "en_US.ISO-8859-1")
	if got := detectEncodingFromLocale(); got != "ISO-8859-1" {
		t.Fatalf("detectEncodingFromLocale() = %s, want ISO-8859-1", got)
	}

	t.Setenv("LC_ALL", "en_US.UTF-8")
	if got := detectEncodingFromLocale(); got != "UTF-8" {
		t.Fatalf("detectEncodingFromLocale() = %s, want UTF-8", got)
	}
}
