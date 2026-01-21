package ui

import (
	"fmt"
	"io"
)

// ClearScreen clears the terminal screen.
func ClearScreen(out io.Writer) {
	_, _ = fmt.Fprint(out, "\033[2J")
}

// MoveCursor moves the cursor to the given row and column (1-based).
func MoveCursor(out io.Writer, row, col int) {
	if row < 1 {
		row = 1
	}
	if col < 1 {
		col = 1
	}
	_, _ = fmt.Fprintf(out, "\033[%d;%dH", row, col)
}

// ClearLine clears the current line from the cursor to the end.
func ClearLine(out io.Writer) {
	_, _ = fmt.Fprint(out, "\033[K")
}

// ClearScreenAndHome clears the screen and moves cursor to home.
func ClearScreenAndHome(out io.Writer) {
	ClearScreen(out)
	MoveCursor(out, 1, 1)
}

// SetColor sets a basic ANSI color (0-7) for foreground.
func SetColor(out io.Writer, color int) {
	if color < 0 {
		color = 0
	}
	if color > 7 {
		color = 7
	}
	_, _ = fmt.Fprintf(out, "\033[3%dm", color)
}

// SetStyle sets text styles: bold, underline, italic.
func SetStyle(out io.Writer, bold, underline, italic bool) {
	// Reset first, then apply requested styles.
	_, _ = fmt.Fprint(out, "\033[0m")
	if bold {
		_, _ = fmt.Fprint(out, "\033[1m")
	}
	if underline {
		_, _ = fmt.Fprint(out, "\033[4m")
	}
	if italic {
		_, _ = fmt.Fprint(out, "\033[3m")
	}
}

// SetColor256 sets a 256-color ANSI foreground color.
func SetColor256(out io.Writer, color int) {
	if color < 0 {
		color = 0
	}
	if color > 255 {
		color = 255
	}
	_, _ = fmt.Fprintf(out, "\033[38;5;%dm", color)
}

// SetTrueColor sets a truecolor RGB foreground color.
func SetTrueColor(out io.Writer, r, g, b int) {
	if r < 0 {
		r = 0
	}
	if r > 255 {
		r = 255
	}
	if g < 0 {
		g = 0
	}
	if g > 255 {
		g = 255
	}
	if b < 0 {
		b = 0
	}
	if b > 255 {
		b = 255
	}
	_, _ = fmt.Fprintf(out, "\033[38;2;%d;%d;%dm", r, g, b)
}
