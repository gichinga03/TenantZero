// Package terminal provides raw-mode line input and styled output for TenantZero.
package terminal

import (
	"errors"
	"io"
	"os"

	"golang.org/x/term"
)

// ErrInterrupt is returned by ReadLine when the user presses Ctrl+C.
var ErrInterrupt = errors.New("interrupt")

// ReadLine reads one line from stdin using raw terminal mode.
//
// Features:
//   - Displays prompt before input
//   - Up/Down arrows navigate history (passed in; caller owns the slice)
//   - Backspace deletes the last character
//   - Ctrl+C returns ("", ErrInterrupt)
//   - Ctrl+D on empty input returns ("", io.EOF)
//   - Tab is a no-op (completion deferred)
//
// History is not modified by ReadLine.
func ReadLine(prompt string, history []string) (string, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer term.Restore(fd, oldState)

	os.Stdout.WriteString(prompt)

	var buf []rune
	histIdx := len(history) // one past the end → not navigating
	savedBuf := ""          // snapshot of current line before history navigation

	// eraseCurrentLine clears the visible line and reprints prompt+buf.
	eraseCurrentLine := func() {
		// Move to start of line, clear to end, reprint.
		os.Stdout.WriteString("\r\033[K")
		os.Stdout.WriteString(prompt)
		os.Stdout.WriteString(string(buf))
	}

	b := make([]byte, 4)
	for {
		n, err := os.Stdin.Read(b)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", io.EOF
			}
			return "", err
		}

		switch {
		case n == 1 && b[0] == '\r' || n == 1 && b[0] == '\n':
			// Enter
			os.Stdout.WriteString("\r\n")
			return string(buf), nil

		case n == 1 && b[0] == 3:
			// Ctrl+C
			os.Stdout.WriteString("^C\r\n")
			return "", ErrInterrupt

		case n == 1 && b[0] == 4:
			// Ctrl+D
			if len(buf) == 0 {
				os.Stdout.WriteString("\r\n")
				return "", io.EOF
			}

		case n == 1 && b[0] == 9:
			// Tab — no-op

		case n == 1 && (b[0] == 127 || b[0] == 8):
			// Backspace
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				eraseCurrentLine()
			}

		case n >= 3 && b[0] == 27 && b[1] == '[':
			// ANSI escape sequences
			switch b[2] {
			case 'A': // Up arrow
				if histIdx > 0 {
					if histIdx == len(history) {
						savedBuf = string(buf)
					}
					histIdx--
					buf = []rune(history[histIdx])
					eraseCurrentLine()
				}
			case 'B': // Down arrow
				if histIdx < len(history) {
					histIdx++
					if histIdx == len(history) {
						buf = []rune(savedBuf)
					} else {
						buf = []rune(history[histIdx])
					}
					eraseCurrentLine()
				}
			}

		case n == 1 && b[0] >= 32:
			// Printable ASCII
			buf = append(buf, rune(b[0]))
			os.Stdout.Write(b[:1])

		default:
			// Multi-byte UTF-8 or unhandled sequences — append and echo.
			if n > 0 && b[0] >= 0xC0 {
				buf = append(buf, []rune(string(b[:n]))...)
				os.Stdout.Write(b[:n])
			}
		}
	}
}