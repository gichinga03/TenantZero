package engine

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/0x4D67/tenantzero/shared/schema"
)

// FS provides Windows-flavoured virtual filesystem operations over the lab's
// static FileSystem and FileContents maps. All path operations are
// case-insensitive and backslash/forward-slash equivalent.
type FS struct {
	lab   *schema.Lab
	state *schema.SessionState
}

// NewFS constructs an FS bound to the given lab and state.
func NewFS(lab *schema.Lab, state *schema.SessionState) *FS {
	return &FS{lab: lab, state: state}
}

// Ls returns a Windows dir-style listing for the given path, or state.Cwd if
// path is empty. Directories are listed before files. Sizes and timestamps are
// deterministic fakes seeded from the entry name so they remain stable across
// calls.
//
//	C:\Users\Public>
//	 Volume in drive C has no label.
//	 Directory of C:\Users\Public
//
//	11/22/2024  09:14 AM    <DIR>          Desktop
//	11/22/2024  09:14 AM             1,024 notes.txt
func (f *FS) Ls(p string) string {
	if p == "" {
		p = f.state.CWD
	}
	norm := normPath(p)

	entries, ok := f.lab.FileSystem[norm]
	if !ok {
		return "The system cannot find the path specified."
	}

	var dirs, files []string
	for _, e := range entries {
		child := normPath(norm + `\` + e)
		if _, isDir := f.lab.FileSystem[child]; isDir {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}
	sort.Strings(dirs)
	sort.Strings(files)

	var sb strings.Builder
	displayPath := toWinPath(norm)
	sb.WriteString(" Volume in drive C has no label.\n")
	sb.WriteString(fmt.Sprintf(" Directory of %s\n\n", displayPath))

	writeEntry := func(name string, isDir bool) {
		ts := fakeTimestamp(name)
		if isDir {
			sb.WriteString(fmt.Sprintf("%s    <DIR>          %s\n", ts, name))
		} else {
			size := fakeSize(name)
			sb.WriteString(fmt.Sprintf("%s    %14s %s\n", ts, formatSize(size), name))
		}
	}

	for _, d := range dirs {
		writeEntry(d, true)
	}
	for _, file := range files {
		writeEntry(file, false)
	}

	total := len(dirs) + len(files)
	sb.WriteString(fmt.Sprintf("\n               %d File(s)\n", len(files)))
	sb.WriteString(fmt.Sprintf("               %d Dir(s)\n", total))
	return sb.String()
}

// Cd validates that target exists in the FileSystem map and returns the
// resolved absolute path. Returns an error string (not an error value) on
// failure, matching Windows cmd.exe behaviour.
func (f *FS) Cd(target string) (string, error) {
	if target == "" {
		return f.state.CWD, nil
	}

	resolved := resolvePath(f.state.CWD, target)
	norm := normPath(resolved)

	if _, ok := f.lab.FileSystem[norm]; !ok {
		return "", fmt.Errorf("The system cannot find the path specified.")
	}

	return toWinPath(norm), nil
}

// Cat returns the contents of the file at p. Access rules:
//   - DA-only files (marked in FileContents) require session user to be DA.
//   - System32 / Windows paths without an explicit FileContents entry return
//     "Access is denied." to simulate protected system files.
//   - Everything else without a FileContents entry returns "The system cannot
//     find the file specified."
func (f *FS) Cat(p string) string {
	if p == "" {
		return "The system cannot find the file specified."
	}

	norm := normPath(p)

	content, ok := f.lab.FileContents[norm]
	if !ok {
		if isSystemPath(norm) {
			return "Access is denied."
		}
		return "The system cannot find the file specified."
	}

	return content
}

// --- path helpers ---

// normPath converts a display path to a canonical lowercase, backslash form
// used as map keys. Strips trailing backslash unless root.
func normPath(p string) string {
	// unify separators
	p = strings.ReplaceAll(p, "/", `\`)
	p = strings.ToLower(p)
	p = path.Clean(strings.ReplaceAll(p, `\`, `/`))
	p = strings.ReplaceAll(p, "/", `\`)
	// path.Clean strips leading \\; restore drive letter root
	if !strings.HasPrefix(p, `c:\`) && !strings.HasPrefix(p, `c:`) && !strings.HasPrefix(p, `~`) {
		p = `c:\` + strings.TrimPrefix(p, `\`)
	}
	p = strings.TrimRight(p, `\`)
	if p == "c:" {
		p = `c:\`
	}
	return p
}

// resolvePath resolves target relative to cwd. Handles "..", absolute, and
// relative paths.
func resolvePath(cwd, target string) string {
	t := strings.ReplaceAll(target, "/", `\`)
	// absolute path: starts with drive letter or backslash
	if len(t) >= 2 && t[1] == ':' {
		return t
	}
	if strings.HasPrefix(t, `\`) {
		return `C:` + t
	}

	// relative: build from cwd
	base := strings.ReplaceAll(cwd, "/", `\`)
	parts := strings.Split(strings.TrimSuffix(base, `\`), `\`)
	for _, seg := range strings.Split(t, `\`) {
		switch seg {
		case "", ".":
			// no-op
		case "..":
			if len(parts) > 1 {
				parts = parts[:len(parts)-1]
			}
		default:
			parts = append(parts, seg)
		}
	}
	return strings.Join(parts, `\`)
}

// toWinPath converts a normalised key back to a display path with proper
// capitalisation convention (C:\ prefix).
func toWinPath(norm string) string {
	if norm == `c:\` || norm == "c:" {
		return `C:\`
	}
	if strings.HasPrefix(norm, `c:\`) {
		return `C:\` + norm[3:]
	}
	return norm
}

// isSystemPath returns true for paths under Windows or System32 — files
// without explicit content entries get "Access is denied." there.
func isSystemPath(norm string) bool {
	return strings.Contains(norm, `windows\`) ||
		strings.Contains(norm, `system32\`) ||
		strings.Contains(norm, `syswow64\`)
}

// isDA returns true if the current session user is a Domain Admin.
func isDA(state *schema.SessionState, lab *schema.Lab) bool {
	const daGroup = "domain admins"
	for _, g := range lab.Groups {
		if strings.EqualFold(g.Name, daGroup) {
			for _, m := range g.Members {
				if strings.EqualFold(m, state.User) {
					return true
				}
			}
		}
	}
	return false
}

// --- fake-but-deterministic metadata ---

// fakeTimestamp returns a stable fake Windows dir timestamp for a given name.
// Seed is the sum of runes mod a fixed offset so the same name always produces
// the same date.
func fakeTimestamp(name string) string {
	var n int
	for _, r := range name {
		n += int(r)
	}
	base := time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)
	offset := time.Duration(n%180) * 24 * time.Hour
	t := base.Add(offset)
	return t.Format("01/02/2006  03:04 PM")
}

// fakeSize returns a deterministic fake file size in bytes.
func fakeSize(name string) int {
	var n int
	for _, r := range name {
		n += int(r)
	}
	return 512 + (n%15)*256
}

// formatSize formats an integer size with comma separators, e.g. 1,024.
func formatSize(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return string(out)
}