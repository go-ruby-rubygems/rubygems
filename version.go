// Package rubygems is a pure-Go (CGO=0), MRI-faithful re-implementation of the
// pure-compute core of Ruby's RubyGems: Gem::Version, Gem::Requirement,
// Gem::Dependency and a usable subset of Gem::Specification.
//
// It matches MRI's bundled RubyGems byte-for-byte on the two algorithms that
// the rest of the gem ecosystem depends on:
//
//   - Version#<=>           (the segment-comparison / canonicalization rules)
//   - Requirement#satisfied_by?  (including the pessimistic "~>" bound math)
//
// The gem index fetch, gem install, gemspec evaluation and require-time
// activation are deliberately out of scope: those are host-side concerns that
// touch the network and the filesystem. This package is the comparable,
// in-memory model everything else is built on. See doc.go for the boundary.
package rubygems

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// VersionPattern is the body of RubyGems' Gem::Version::VERSION_PATTERN. It
// matches a leading numeric segment, optional dotted alphanumeric segments, and
// an optional dash-prefixed build tail (which RubyGems rewrites to ".pre.").
const VersionPattern = `[0-9]+(?:\.[0-9a-zA-Z]+)*(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?`

// anchoredVersionPattern mirrors Gem::Version::ANCHORED_VERSION_PATTERN: an
// optional VERSION_PATTERN surrounded by optional whitespace.
var anchoredVersionPattern = regexp.MustCompile(`\A\s*(` + VersionPattern + `)?\s*\z`)

// segmentScan mirrors the /\d+|[a-z]+/i scan RubyGems uses to split a version
// string into segments.
var segmentScan = regexp.MustCompile(`[0-9]+|[a-zA-Z]+`)

// letterRE matches the presence of any ASCII letter, used for prerelease
// detection (Gem::Version#prerelease?).
var letterRE = regexp.MustCompile(`[a-zA-Z]`)

// trailingZeroRE strips trailing ".0"/"."/"0" runs anchored to a letter or dot,
// mirroring the first sub in Gem::Version#canonical_segments:
//
//	@version.sub(/(?<=[a-zA-Z.])[.0]+\z/, "")
//
// Go's regexp has no lookbehind, so the anchor character is captured in group 1
// and re-emitted by the replacement.
var trailingZeroRE = regexp.MustCompile(`([a-zA-Z.])[.0]+\z`)

// leadingZeroRE removes "0" segments before the first letter of a prerelease,
// mirroring:
//
//	canonical_version.sub!(/(?<=\.|\A)[0.]+(?=[a-zA-Z])/, "")
//
// The (?<=\.|\A) lookbehind is emulated with an optional captured leading dot,
// and (?=[a-zA-Z]) with a captured following letter.
var leadingZeroRE = regexp.MustCompile(`(\.|\A)[0.]+([a-zA-Z])`)

// segment is one parsed version segment: either a number (str == "") or a
// string (num == 0, str set). RubyGems sorts numbers above strings.
type segment struct {
	num int64
	str string
}

func (s segment) isString() bool { return s.str != "" }

// Version is a comparable RubyGems version (Gem::Version).
type Version struct {
	version           string // canonicalized string form (with "-" -> ".pre.")
	segments          []segment
	canonicalSegments []segment
	prerelease        bool
}

// CorrectVersion reports whether s is a well-formed version string, mirroring
// Gem::Version.correct?. A nil-equivalent (handled by callers) and the empty
// string are accepted by RubyGems; here the empty string matches the anchored
// pattern's optional group.
func CorrectVersion(s string) bool {
	return anchoredVersionPattern.MatchString(s)
}

// NewVersion constructs a Version from s, mirroring Gem::Version#initialize. It
// returns an error (rather than raising ArgumentError) for malformed input.
func NewVersion(s string) (*Version, error) {
	if !CorrectVersion(s) {
		return nil, fmt.Errorf("Malformed version number string %s", s)
	}

	// Empty / all-whitespace becomes "0" (Gem::Version#initialize).
	if strings.TrimSpace(s) == "" {
		s = "0"
	}

	v := strings.TrimSpace(s)
	v = strings.ReplaceAll(v, "-", ".pre.")

	ver := &Version{version: v}
	ver.segments = partitionSegments(v)
	ver.prerelease = letterRE.MatchString(v)
	ver.canonicalSegments = computeCanonicalSegments(v, ver.prerelease)
	return ver, nil
}

// MustVersion is like NewVersion but panics on malformed input. It is a
// convenience for tests and for callers that know the input is well-formed.
func MustVersion(s string) *Version {
	v, err := NewVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}

// partitionSegments mirrors Gem::Version#partition_segments: scan numbers and
// letter-runs, mapping numeric runs to integers and letter runs to strings.
func partitionSegments(ver string) []segment {
	matches := segmentScan.FindAllString(ver, -1)
	segs := make([]segment, 0, len(matches))
	for _, m := range matches {
		if m[0] >= '0' && m[0] <= '9' {
			n, _ := strconv.ParseInt(m, 10, 64)
			segs = append(segs, segment{num: n})
		} else {
			segs = append(segs, segment{str: m})
		}
	}
	return segs
}

// computeCanonicalSegments mirrors Gem::Version#canonical_segments: drop
// trailing zero segments (anchored to a dot or letter) and, for prereleases,
// drop zero segments before the first letter.
func computeCanonicalSegments(version string, prerelease bool) []segment {
	canonical := trailingZeroRE.ReplaceAllString(version, "$1")
	if prerelease {
		canonical = leadingZeroRE.ReplaceAllString(canonical, "$1$2")
	}
	return partitionSegments(canonical)
}

// String returns the version string (Gem::Version#to_s / #version).
func (v *Version) String() string { return v.version }

// Segments returns a copy of the raw parsed segments. Numeric segments are
// returned as int64; string segments as string (Gem::Version#segments).
func (v *Version) Segments() []any {
	out := make([]any, len(v.segments))
	for i, s := range v.segments {
		if s.isString() {
			out[i] = s.str
		} else {
			out[i] = s.num
		}
	}
	return out
}

// CanonicalSegments returns a copy of the canonical segments
// (Gem::Version#canonical_segments).
func (v *Version) CanonicalSegments() []any {
	out := make([]any, len(v.canonicalSegments))
	for i, s := range v.canonicalSegments {
		if s.isString() {
			out[i] = s.str
		} else {
			out[i] = s.num
		}
	}
	return out
}

// Prerelease reports whether the version contains a letter
// (Gem::Version#prerelease?).
func (v *Version) Prerelease() bool { return v.prerelease }

// Compare returns -1, 0, or 1 as v is less than, equal to, or greater than
// other, faithfully porting Gem::Version#<=> over canonical segments.
func (v *Version) Compare(other *Version) int {
	if v.version == other.version || canonSegmentsEqual(v.canonicalSegments, other.canonicalSegments) {
		return 0
	}

	lh := v.canonicalSegments
	rh := other.canonicalSegments
	lhsize := len(lh)
	rhsize := len(rh)
	limit := lhsize
	if rhsize < limit {
		limit = rhsize
	}

	i := 0
	for i < limit {
		lhs := lh[i]
		rhs := rh[i]
		i++

		if segEqual(lhs, rhs) {
			continue
		}
		if lhs.isString() && !rhs.isString() {
			return -1
		}
		if !lhs.isString() && rhs.isString() {
			return 1
		}
		return segCompare(lhs, rhs)
	}

	// One side is a strict prefix of the other (the loop consumed every shared
	// segment without deciding). Canonical segment lists never end in a numeric
	// zero (canonical_segments strips trailing zeros), so the first extra
	// segment is always either a string or a non-zero number and decides the
	// order outright — mirroring Gem::Version#<=>, where a string tail makes
	// that side smaller and a non-zero numeric tail makes it larger.
	if i >= lhsize {
		// Left exhausted; the right tail's first segment is the larger side's.
		rhs := rh[i]
		if rhs.isString() {
			return 1
		}
		return -1
	}
	// Right exhausted; the left tail's first segment decides.
	lhs := lh[i]
	if lhs.isString() {
		return -1
	}
	return 1
}

func segEqual(a, b segment) bool {
	if a.isString() != b.isString() {
		return false
	}
	if a.isString() {
		return a.str == b.str
	}
	return a.num == b.num
}

// segCompare compares two same-typed segments. It is only called from Compare
// on segments already known to be unequal, so for numbers the result is never
// zero; strings delegate to lexical comparison.
func segCompare(a, b segment) int {
	if a.isString() {
		return strings.Compare(a.str, b.str)
	}
	if a.num < b.num {
		return -1
	}
	return 1
}

func canonSegmentsEqual(a, b []segment) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !segEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// Equal reports value equality using the comparison ordering (==), i.e.
// Compare(other) == 0. Note "1.0" == "1.0.0".
func (v *Version) Equal(other *Version) bool { return v.Compare(other) == 0 }

// Eql reports whether v and other are eql?: equal AND specified to the same
// precision (Gem::Version#eql?). "1.0" is not eql? "1".
func (v *Version) Eql(other *Version) bool { return v.version == other.version }

// HashKey returns a stable key derived from the canonical segments, suitable
// for use as a Go map key. Two versions that are Eql? share a HashKey, mirroring
// the contract of Gem::Version#hash (which hashes canonical_segments).
func (v *Version) HashKey() string {
	var b strings.Builder
	for _, s := range v.canonicalSegments {
		if s.isString() {
			b.WriteByte('s')
			b.WriteString(s.str)
		} else {
			b.WriteByte('n')
			b.WriteString(strconv.FormatInt(s.num, 10))
		}
		b.WriteByte('.')
	}
	return b.String()
}

// Bump returns a new version where the next-to-last numeric segment is
// incremented and the rest dropped (Gem::Version#bump). Prerelease tails are
// ignored: 5.3.1 -> 5.4, 5.3.1.b.2 -> 5.4.
func (v *Version) Bump() *Version {
	segs := append([]segment(nil), v.segments...)
	// drop trailing string segments
	for anyString(segs) {
		segs = segs[:len(segs)-1]
	}
	if len(segs) > 1 {
		segs = segs[:len(segs)-1]
	}
	segs[len(segs)-1].num++
	return MustVersion(joinSegments(segs))
}

// Release returns the release version (Gem::Version#release): for a prerelease,
// drop all string segments; otherwise return the version itself.
func (v *Version) Release() *Version {
	if !v.prerelease {
		return v
	}
	segs := append([]segment(nil), v.segments...)
	for anyString(segs) {
		segs = segs[:len(segs)-1]
	}
	return MustVersion(joinSegments(segs))
}

// ApproximateRecommendation returns a "~>" requirement string recommended for
// this version (Gem::Version#approximate_recommendation). Two segments are
// always used; a prerelease appends ".a".
func (v *Version) ApproximateRecommendation() string {
	segs := append([]segment(nil), v.segments...)
	for anyString(segs) {
		segs = segs[:len(segs)-1]
	}
	for len(segs) > 2 {
		segs = segs[:len(segs)-1]
	}
	for len(segs) < 2 {
		segs = append(segs, segment{num: 0})
	}
	rec := "~> " + joinSegments(segs)
	if v.prerelease {
		rec += ".a"
	}
	return rec
}

func anyString(segs []segment) bool {
	for _, s := range segs {
		if s.isString() {
			return true
		}
	}
	return false
}

// joinSegments dots together segments for Bump/Release/ApproximateRecommendation.
// All callers strip every string segment first, so only numeric segments reach
// here.
func joinSegments(segs []segment) string {
	parts := make([]string, len(segs))
	for i, s := range segs {
		parts[i] = strconv.FormatInt(s.num, 10)
	}
	return strings.Join(parts, ".")
}
