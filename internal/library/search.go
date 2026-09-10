package library

import (
	"strings"
	"time"
)

// Query filters a set of samples. The zero value matches everything; each
// non-zero field narrows the results further (fields are ANDed together).
type Query struct {
	// Name matches (case-insensitively) against the sample's path, so a
	// query like "kick" matches both a filename and a containing folder.
	Name string

	// Format, if set, restricts results to that exact format.
	Format Format

	// MinDuration and MaxDuration, if non-zero, bound the sample's
	// duration. A sample with unknown duration (HasAudioMetadata false)
	// never matches a query that sets either bound.
	MinDuration time.Duration
	MaxDuration time.Duration
}

// Match reports whether s satisfies q.
func (q Query) Match(s Sample) bool {
	if q.Name != "" && !strings.Contains(strings.ToLower(s.Path), strings.ToLower(q.Name)) {
		return false
	}
	if q.Format != "" && s.Format != q.Format {
		return false
	}
	if q.MinDuration > 0 || q.MaxDuration > 0 {
		if !s.HasAudioMetadata() {
			return false
		}
		if q.MinDuration > 0 && s.Duration < q.MinDuration {
			return false
		}
		if q.MaxDuration > 0 && s.Duration > q.MaxDuration {
			return false
		}
	}
	return true
}

// IsZero reports whether q has no constraints set (and so matches
// everything).
func (q Query) IsZero() bool {
	return q.Name == "" && q.Format == "" && q.MinDuration == 0 && q.MaxDuration == 0
}

// Filter returns the samples in all that match q, preserving order.
func Filter(all []Sample, q Query) []Sample {
	if q.IsZero() {
		return all
	}
	matches := make([]Sample, 0, len(all))
	for _, s := range all {
		if q.Match(s) {
			matches = append(matches, s)
		}
	}
	return matches
}
