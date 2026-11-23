// Package journal records session commands to disk and exports markdown reports.
package journal

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Entry is a single recorded command and its output.
type Entry struct {
	Sequence  int       `json:"sequence"`
	Input     string    `json:"input"`
	Output    string    `json:"output"`
	Machine   string    `json:"machine"`
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
}

// Journal records entries for one lab session.
type Journal struct {
	path    string
	labID   string
	entries []Entry
	f       *os.File
}

// New creates or opens the journal file for labID on the current date.
// Path: journalDir/<labID>-<YYYY-MM-DD>.log (append mode).
func New(journalDir, labID string) (*Journal, error) {
	if err := os.MkdirAll(journalDir, 0o700); err != nil {
		return nil, fmt.Errorf("journal: mkdir %s: %w", journalDir, err)
	}
	date := time.Now().Format("2006-01-02")
	path := filepath.Join(journalDir, fmt.Sprintf("%s-%s.log", labID, date))

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("journal: open %s: %w", path, err)
	}

	j := &Journal{path: path, labID: labID, f: f}

	// Replay existing entries so sequence numbers continue correctly.
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		dec := json.NewDecoder(bytesReader(data))
		for dec.More() {
			var e Entry
			if err := dec.Decode(&e); err == nil {
				j.entries = append(j.entries, e)
			}
		}
	}
	return j, nil
}

// bytesReader wraps a byte slice in an io.Reader (stdlib only).
func bytesReader(b []byte) io.Reader {
	return &byteReader{b: b}
}

type byteReader struct {
	b   []byte
	pos int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.pos:])
	r.pos += n
	return n, nil
}

// Append adds an entry to the journal and flushes it to disk immediately.
// The entry's Sequence field is set by the journal.
func (j *Journal) Append(e Entry) error {
	e.Sequence = len(j.entries) + 1
	j.entries = append(j.entries, e)

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("journal: marshal entry: %w", err)
	}
	data = append(data, '\n')
	if _, err := j.f.Write(data); err != nil {
		return fmt.Errorf("journal: write entry: %w", err)
	}
	return nil
}

// Close flushes and closes the underlying file.
func (j *Journal) Close() error {
	return j.f.Close()
}

// ExportMarkdown writes the full journal as a markdown report skeleton to w.
func (j *Journal) ExportMarkdown(w io.Writer) error {
	date := time.Now().Format("2006-01-02")
	_, err := fmt.Fprintf(w, "# Lab Journal — %s\n\nDate: %s\n\n## Commands\n\n", j.labID, date)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(w, "| # | Machine | User | Input | Timestamp |")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, "|---|---------|------|-------|-----------|")
	if err != nil {
		return err
	}

	for _, e := range j.entries {
		_, err = fmt.Fprintf(w, "| %d | %s | %s | `%s` | %s |\n",
			e.Sequence,
			e.Machine,
			e.Username,
			e.Input,
			e.Timestamp.Format(time.RFC3339),
		)
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintln(w, "\n## Notes\n\n<!-- Add analysis here -->")
	return err
}