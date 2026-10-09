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

func TestWindowNumberConversionEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    int
		wantErr bool
	}{
		{"lowercase a", "a", 10, false},
		{"lowercase z", "z", 35, false},
		{"lowercase b", "b", 11, false},
		{"invalid char", "@", -1, true},
		{"invalid number", "36", -1, true},
		{"negative", "-1", -1, true},
		{"two chars", "AB", -1, true},
		{"empty", "", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := windowStringToNumber(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("windowStringToNumber(%q) error = %v, wantErr %v", tt.s, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("windowStringToNumber(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestWindowGetPTYProcess(t *testing.T) {
	w := &Window{}
	ptyProc := w.GetPTYProcess()
	if ptyProc != nil {
		t.Error("Expected nil PTYProcess")
	}
}

func TestWindowSetPTYProcess(t *testing.T) {
	w := &Window{}
	if w.Pid != 0 || w.PtsPath != "" {
		t.Error("Expected empty Pid and PtsPath")
	}
}

func TestWindowKill(t *testing.T) {
	w := &Window{}
	err := w.Kill()
	if err != nil {
		t.Errorf("Window.Kill() with nil PTYProcess should not error, got %v", err)
	}
}

func TestWindowIsAlive(t *testing.T) {
	w := &Window{}
	alive := w.IsAlive()
	if alive {
		t.Error("Window with nil PTYProcess should not be alive")
	}
}
