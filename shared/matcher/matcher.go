// Package matcher implements three-level command matching for the TenantZero
// lab engine: exact, wildcard, and fuzzy (Levenshtein). It is used by both
// the server-side command dispatcher and the offline client runner.
package matcher

import (
	"strings"
	"unicode"
)

// MatchLevel describes how closely an input matched a pattern.
type MatchLevel int

const (
	// MatchExact means the normalised input equalled the normalised pattern.
	MatchExact MatchLevel = iota
	// MatchWildcard means the input matched a pattern containing one or more
	// '*' tokens, each of which captured exactly one whitespace-delimited token.
	MatchWildcard
	// MatchFuzzy means the Levenshtein distance between the normalised input
	// and the normalised pattern was within the configured threshold.
	MatchFuzzy
	// MatchNone means no pattern could be matched at any level.
	MatchNone
)

// MatchResult is returned by [Matcher.Match] and describes the winning match.
type MatchResult struct {
	// Pattern is the original pattern string that produced the match.
	Pattern string
	// Level is the match quality: Exact > Wildcard > Fuzzy > None.
	Level MatchLevel
	// Wildcards holds the token values captured by each '*' in the pattern,
	// in left-to-right order. It is nil when Level is not MatchWildcard.
	Wildcards []string
}

// Matcher holds a set of command patterns and configuration for fuzzy matching.
// The zero value is not safe to use; construct with [NewMatcher].
type Matcher struct {
	patterns  []string
	threshold int // maximum Levenshtein distance for a fuzzy match
}

// NewMatcher constructs a Matcher pre-loaded with patterns. threshold is the
// maximum edit distance (inclusive) for fuzzy matching; pass 2 for the
// recommended default.
func NewMatcher(patterns []string, threshold int) *Matcher {
	m := &Matcher{threshold: threshold}
	for _, p := range patterns {
		m.patterns = append(m.patterns, p)
	}
	return m
}

// AddPattern appends a single pattern to the matcher at runtime.
func (m *Matcher) AddPattern(pattern string) {
	m.patterns = append(m.patterns, pattern)
}

// Match attempts to match input against all registered patterns, returning the
// best result. Priority: Exact > Wildcard > Fuzzy. Among patterns at the same
// level the most specific one wins: fewest wildcards first, then shortest
// pattern string.
func (m *Matcher) Match(input string) MatchResult {
	norm := normalize(input)

	// --- pass 1: exact ---
	for _, p := range m.patterns {
		if normalize(p) == norm {
			return MatchResult{Pattern: p, Level: MatchExact}
		}
	}

	// --- pass 2: wildcard ---
	var bestWild *MatchResult
	for _, p := range m.patterns {
		if !strings.Contains(p, "*") {
			continue
		}
		if caps, ok := wildcardMatch(normalize(p), norm); ok {
			candidate := MatchResult{Pattern: p, Level: MatchWildcard, Wildcards: caps}
			if bestWild == nil || isMoreSpecific(candidate, *bestWild) {
				c := candidate
				bestWild = &c
			}
		}
	}
	if bestWild != nil {
		return *bestWild
	}

	// --- pass 3: fuzzy ---
	var bestFuzzy *MatchResult
	bestDist := m.threshold + 1
	for _, p := range m.patterns {
		d := levenshtein(norm, normalize(p))
		if d > m.threshold {
			continue
		}
		if d < bestDist {
			bestDist = d
			candidate := MatchResult{Pattern: p, Level: MatchFuzzy}
			bestFuzzy = &candidate
		} else if d == bestDist && bestFuzzy != nil {
			// tie-break: shorter pattern wins
			if len(p) < len(bestFuzzy.Pattern) {
				candidate := MatchResult{Pattern: p, Level: MatchFuzzy}
				bestFuzzy = &candidate
			}
		}
	}
	if bestFuzzy != nil {
		return *bestFuzzy
	}

	return MatchResult{Level: MatchNone}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// normalize trims leading/trailing whitespace, collapses all internal
// whitespace runs to a single space, and lowercases the result.
func normalize(s string) string {
	var b strings.Builder
	prevSpace := true // treat leading edge as "previous was space" to trim
	for _, r := range strings.TrimFunc(s, unicode.IsSpace) {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
			}
			prevSpace = true
		} else {
			b.WriteRune(unicode.ToLower(r))
			prevSpace = false
		}
	}
	return b.String()
}

// wildcardMatch checks whether the normalised pattern (which may contain '*'
// tokens) covers the normalised input exactly. It returns the captured values
// for each '*' on success.
//
// Rules:
//   - '*' matches exactly one whitespace-delimited token in the input.
//   - Non-wildcard tokens must match the corresponding input token verbatim.
//   - The match must be exhaustive: leftover input or pattern tokens fail.
func wildcardMatch(pattern, input string) ([]string, bool) {
	pp := strings.Fields(pattern)
	ip := strings.Fields(input)
	if len(pp) != len(ip) {
		return nil, false
	}
	var caps []string
	for i, pt := range pp {
		if pt == "*" {
			caps = append(caps, ip[i])
		} else if pt != ip[i] {
			return nil, false
		}
	}
	return caps, true
}

// isMoreSpecific returns true if candidate should beat current at the same
// match level. Tie-break order: fewer wildcards → shorter pattern string.
func isMoreSpecific(candidate, current MatchResult) bool {
	cw := strings.Count(candidate.Pattern, "*")
	ew := strings.Count(current.Pattern, "*")
	if cw != ew {
		return cw < ew
	}
	return len(candidate.Pattern) < len(current.Pattern)
}

// levenshtein computes the edit distance between two strings using the classic
// Wagner-Fischer DP algorithm. Space: O(min(len(a), len(b))).
func levenshtein(a, b string) int {
	ra := []rune(a)
	rb := []rune(b)
	// ensure |ra| <= |rb| to minimise the row allocation
	if len(ra) > len(rb) {
		ra, rb = rb, ra
	}
	n := len(ra)
	m := len(rb)
	prev := make([]int, n+1)
	curr := make([]int, n+1)
	for i := range prev {
		prev[i] = i
	}
	for j := 1; j <= m; j++ {
		curr[0] = j
		for i := 1; i <= n; i++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			del := prev[i] + 1
			ins := curr[i-1] + 1
			sub := prev[i-1] + cost
			curr[i] = min3(del, ins, sub)
		}
		prev, curr = curr, prev
	}
	return prev[n]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}