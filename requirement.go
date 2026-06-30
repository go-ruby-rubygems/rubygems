package rubygems

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// requirementPattern mirrors Gem::Requirement::PATTERN: an optional operator
// followed by a VERSION_PATTERN, with optional surrounding whitespace.
var requirementPattern = regexp.MustCompile(
	`\A\s*(=|!=|>|<|>=|<=|~>)?\s*(` + VersionPattern + `)\s*\z`)

// constraint is one parsed [op, version] pair of a requirement.
type constraint struct {
	op      string
	version *Version
}

// Requirement is a set of one or more version constraints (Gem::Requirement).
type Requirement struct {
	requirements []constraint
}

// defaultConstraintString is the canonical form of the default ">= 0".
const defaultConstraintString = ">= 0"

// ParseConstraint parses a single requirement string into an [op, version]
// pair, mirroring Gem::Requirement.parse. A bare version defaults to "=".
func ParseConstraint(s string) (op string, version *Version, err error) {
	m := requirementPattern.FindStringSubmatch(s)
	if m == nil {
		return "", nil, fmt.Errorf("Illformed requirement [%q]", s)
	}
	op = m[1]
	if op == "" {
		op = "="
	}
	// m[2] matched VERSION_PATTERN, so it is always a well-formed version;
	// MustVersion is therefore total here (any panic would be a regex bug).
	return op, MustVersion(m[2]), nil
}

// NewRequirement constructs a Requirement from one or more constraint strings,
// mirroring Gem::Requirement#initialize. nil/duplicate entries are removed and
// an empty set becomes the default ">= 0".
func NewRequirement(constraints ...string) (*Requirement, error) {
	// No constraints at all is the default ">= 0" (Gem::Requirement.new with no
	// args). Note: an *empty string* constraint is NOT the default; like MRI it
	// is an ill-formed requirement and errors in ParseConstraint below.
	//
	// uniq, preserving first-seen order (Array#uniq! semantics).
	seen := map[string]bool{}
	var uniq []string
	for _, c := range constraints {
		if seen[c] {
			continue
		}
		seen[c] = true
		uniq = append(uniq, c)
	}

	if len(uniq) == 0 {
		def, _ := NewVersion("0")
		return &Requirement{requirements: []constraint{{op: ">=", version: def}}}, nil
	}

	reqs := make([]constraint, 0, len(uniq))
	for _, c := range uniq {
		op, v, err := ParseConstraint(c)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, constraint{op: op, version: v})
	}
	return &Requirement{requirements: reqs}, nil
}

// MustRequirement is like NewRequirement but panics on a malformed constraint.
func MustRequirement(constraints ...string) *Requirement {
	r, err := NewRequirement(constraints...)
	if err != nil {
		panic(err)
	}
	return r
}

// DefaultRequirement returns the default requirement ">= 0"
// (Gem::Requirement.default).
func DefaultRequirement() *Requirement { return MustRequirement(defaultConstraintString) }

// DefaultPrereleaseRequirement returns ">= 0.a"
// (Gem::Requirement.default_prerelease).
func DefaultPrereleaseRequirement() *Requirement { return MustRequirement(">= 0.a") }

// SatisfiedBy reports whether version satisfies every constraint
// (Gem::Requirement#satisfied_by?). The "~>" operator uses the pessimistic
// bound math: v >= r && v.release < r.bump.
func (r *Requirement) SatisfiedBy(version *Version) bool {
	for _, c := range r.requirements {
		if !satisfiesOne(c.op, version, c.version) {
			return false
		}
	}
	return true
}

// ops mirrors Gem::Requirement::OPS: the comparison lambda for each operator.
// ParseConstraint only ever emits keys present here, so the lookup in
// satisfiesOne never misses.
var ops = map[string]func(v, r *Version) bool{
	"=":  func(v, r *Version) bool { return v.Compare(r) == 0 },
	"!=": func(v, r *Version) bool { return v.Compare(r) != 0 },
	">":  func(v, r *Version) bool { return v.Compare(r) > 0 },
	"<":  func(v, r *Version) bool { return v.Compare(r) < 0 },
	">=": func(v, r *Version) bool { return v.Compare(r) >= 0 },
	"<=": func(v, r *Version) bool { return v.Compare(r) <= 0 },
	"~>": func(v, r *Version) bool { return v.Compare(r) >= 0 && v.Release().Compare(r.Bump()) < 0 },
}

func satisfiesOne(op string, v, r *Version) bool {
	return ops[op](v, r)
}

// AsList returns the constraints as "op version" strings
// (Gem::Requirement#as_list).
func (r *Requirement) AsList() []string {
	out := make([]string, len(r.requirements))
	for i, c := range r.requirements {
		out[i] = c.op + " " + c.version.String()
	}
	return out
}

// String returns the comma-joined constraint list (Gem::Requirement#to_s).
func (r *Requirement) String() string { return strings.Join(r.AsList(), ", ") }

// None reports whether the requirement is exactly the default ">= 0"
// (Gem::Requirement#none?).
func (r *Requirement) None() bool {
	if len(r.requirements) != 1 {
		return false
	}
	c := r.requirements[0]
	return c.op == ">=" && c.version.String() == "0"
}

// Exact reports whether the requirement is a single "=" constraint
// (Gem::Requirement#exact?).
func (r *Requirement) Exact() bool {
	return len(r.requirements) == 1 && r.requirements[0].op == "="
}

// Specific reports whether the requirement will not always match the latest
// version (Gem::Requirement#specific?).
func (r *Requirement) Specific() bool {
	if len(r.requirements) > 1 {
		return true
	}
	op := r.requirements[0].op
	return op != ">" && op != ">="
}

// Prerelease reports whether any constraint version is a prerelease
// (Gem::Requirement#prerelease?).
func (r *Requirement) Prerelease() bool {
	for _, c := range r.requirements {
		if c.version.Prerelease() {
			return true
		}
	}
	return false
}

// Equal reports requirement equality (Gem::Requirement#==). It mirrors MRI:
// the constraints, sorted by their string form, must match pairwise where the
// versions are compared by VALUE (so ">= 1.2" == ">= 1.2.0"); then, if any
// "~>" is present, the tilde constraints are additionally compared with the
// stricter eql? that also requires identical version precision (so
// "~> 1.2" != "~> 1.2.0").
func (r *Requirement) Equal(other *Requirement) bool {
	ls := r.sortedConstraints()
	rs := other.sortedConstraints()
	if len(ls) != len(rs) {
		return false
	}
	for i := range ls {
		if ls[i].op != rs[i].op || ls[i].version.Compare(rs[i].version) != 0 {
			return false
		}
	}
	if !r.hasTilde() {
		return true
	}
	// Stricter eql? on the tilde constraints: same op + same exact precision.
	lt := r.sortedTildeKeys()
	rt := other.sortedTildeKeys()
	for i := range lt {
		if lt[i] != rt[i] {
			return false
		}
	}
	return true
}

// sortedConstraints returns the constraints sorted by their "op version" string
// form, mirroring MRI's _sorted_requirements (sort_by(&:to_s)).
func (r *Requirement) sortedConstraints() []constraint {
	out := append([]constraint(nil), r.requirements...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].op+" "+out[i].version.String() < out[j].op+" "+out[j].version.String()
	})
	return out
}

func (r *Requirement) hasTilde() bool {
	for _, c := range r.requirements {
		if c.op == "~>" {
			return true
		}
	}
	return false
}

func (r *Requirement) sortedTildeKeys() []string {
	var out []string
	for _, c := range r.sortedConstraints() {
		if c.op == "~>" {
			out = append(out, c.op+" "+c.version.String())
		}
	}
	return out
}
