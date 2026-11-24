// Package terminal provides raw-mode line input and styled output for TenantZero.
package terminal

import (
	"errors"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// ErrInterrupt is returned by ReadLine when the user presses Ctrl+C.
var ErrInterrupt = errors.New("interrupt")

// ReadLine reads one line from stdin using raw terminal mode.
// Supports paste, left/right cursor, home/end, history navigation.
func ReadLine(prompt string, history []string) (string, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer term.Restore(fd, oldState)

	os.Stdout.WriteString(prompt)

	var buf []rune
	cursor := 0
	histIdx := len(history)
	savedBuf := ""

	redraw := func() {
		os.Stdout.WriteString("\r\033[K")
		os.Stdout.WriteString(prompt)
		os.Stdout.WriteString(string(buf))
		// reposition cursor
		if cursor < len(buf) {
			back := len(buf) - cursor
			os.Stdout.WriteString("\033[")
			os.Stdout.WriteString(itoa(back))
			os.Stdout.WriteString("D")
		}
	}

	in := make([]byte, 256)
	for {
		n, err := os.Stdin.Read(in)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", io.EOF
			}
			return "", err
		}

		data := in[:n]
		i := 0
		for i < len(data) {
			b := data[i]

			// Enter
			if b == '\r' || b == '\n' {
				os.Stdout.WriteString("\r\n")
				return string(buf), nil
			}

			// Ctrl+C
			if b == 3 {
				os.Stdout.WriteString("^C\r\n")
				return "", ErrInterrupt
			}

			// Ctrl+D
			if b == 4 {
				if len(buf) == 0 {
					os.Stdout.WriteString("\r\n")
					return "", io.EOF
				}
				i++
				continue
			}

			// Ctrl+A — home
			if b == 1 {
				cursor = 0
				redraw()
				i++
				continue
			}

			// Ctrl+E — end
			if b == 5 {
				cursor = len(buf)
				redraw()
				i++
				continue
			}

			// Ctrl+W — delete word back
			if b == 23 {
				if cursor > 0 {
					start := cursor - 1
					for start > 0 && buf[start-1] != ' ' {
						start--
					}
					buf = append(buf[:start], buf[cursor:]...)
					cursor = start
					redraw()
				}
				i++
				continue
			}

			// Ctrl+U — clear line
			if b == 21 {
				buf = buf[:0]
				cursor = 0
				redraw()
				i++
				continue
			}

			// Backspace
			if b == 127 || b == 8 {
				if cursor > 0 {
					buf = append(buf[:cursor-1], buf[cursor:]...)
					cursor--
					redraw()
				}
				i++
				continue
			}

			// Tab — complete
			if b == 9 {
				completed := tabComplete(string(buf[:cursor]))
				if completed != "" {
					buf = append([]rune(completed), buf[cursor:]...)
					cursor = len(completed)
					redraw()
				}
				i++
				continue
			}

			// ANSI escape sequence
			if b == 27 && i+1 < len(data) && data[i+1] == '[' {
				i += 2
				if i >= len(data) {
					continue
				}
				switch data[i] {
				case 'A': // Up
					if histIdx > 0 {
						if histIdx == len(history) {
							savedBuf = string(buf)
						}
						histIdx--
						buf = []rune(history[histIdx])
						cursor = len(buf)
						redraw()
					}
				case 'B': // Down
					if histIdx < len(history) {
						histIdx++
						if histIdx == len(history) {
							buf = []rune(savedBuf)
						} else {
							buf = []rune(history[histIdx])
						}
						cursor = len(buf)
						redraw()
					}
				case 'C': // Right
					if cursor < len(buf) {
						cursor++
						os.Stdout.WriteString("\033[C")
					}
				case 'D': // Left
					if cursor > 0 {
						cursor--
						os.Stdout.WriteString("\033[D")
					}
				case 'H': // Home
					cursor = 0
					redraw()
				case 'F': // End
					cursor = len(buf)
					redraw()
				case '3': // Delete key (ESC[3~)
					if i+1 < len(data) && data[i+1] == '~' {
						i++
						if cursor < len(buf) {
							buf = append(buf[:cursor], buf[cursor+1:]...)
							redraw()
						}
					}
				}
				i++
				continue
			}

			// Printable / paste: consume all remaining printable bytes in one shot
			if b >= 32 && b < 127 {
				// grab the whole printable run (handles paste bursts)
				start := i
				for i < len(data) && data[i] >= 32 && data[i] < 127 {
					i++
				}
				chunk := []rune(string(data[start:i]))
				// insert at cursor
				buf = append(buf[:cursor], append(chunk, buf[cursor:]...)...)
				cursor += len(chunk)
				redraw()
				continue
			}

			// Multi-byte UTF-8
			if b >= 0xC0 {
				end := i + 1
				for end < len(data) && data[end]&0xC0 == 0x80 {
					end++
				}
				r := []rune(string(data[i:end]))
				buf = append(buf[:cursor], append(r, buf[cursor:]...)...)
				cursor += len(r)
				redraw()
				i = end
				continue
			}

			i++
		}
	}
}

// itoa converts a non-negative int to its decimal string — avoids importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b strings.Builder
	for n > 0 {
		b.WriteByte(byte('0' + n%10))
		n /= 10
	}
	r := []byte(b.String())
	for l, ri := 0, len(r)-1; l < ri; l, ri = l+1, ri-1 {
		r[l], r[ri] = r[ri], r[l]
	}
	return string(r)
}
