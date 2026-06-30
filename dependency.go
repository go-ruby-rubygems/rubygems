package rubygems

import (
	"fmt"
	"strings"
)

// DependencyType is a Gem::Dependency type: runtime or development.
type DependencyType string

const (
	// RuntimeType is the default dependency type (:runtime).
	RuntimeType DependencyType = "runtime"
	// DevelopmentType is the :development dependency type.
	DevelopmentType DependencyType = "development"
)

// Dependency holds a gem name, a Requirement and a type (Gem::Dependency).
type Dependency struct {
	Name        string
	requirement *Requirement
	typ         DependencyType
	prerelease  bool
}

// NewDependency constructs a dependency with the given name, type and
// constraint strings (Gem::Dependency#initialize). An empty constraint set is
// the default ">= 0".
func NewDependency(name string, typ DependencyType, constraints ...string) (*Dependency, error) {
	if name == "" {
		return nil, fmt.Errorf("dependency name must be a String, was %q", name)
	}
	if typ == "" {
		typ = RuntimeType
	}
	if typ != RuntimeType && typ != DevelopmentType {
		return nil, fmt.Errorf("Valid types are [:development, :runtime], not %q", typ)
	}
	req, err := NewRequirement(constraints...)
	if err != nil {
		return nil, err
	}
	return &Dependency{Name: name, requirement: req, typ: typ}, nil
}

// MustDependency is like NewDependency but panics on error.
func MustDependency(name string, typ DependencyType, constraints ...string) *Dependency {
	d, err := NewDependency(name, typ, constraints...)
	if err != nil {
		panic(err)
	}
	return d
}

// Requirement returns the dependency's requirement (Gem::Dependency#requirement).
func (d *Dependency) Requirement() *Requirement { return d.requirement }

// Type returns the dependency type (Gem::Dependency#type). The constructor
// normalizes an empty type to runtime, so the stored value is always set.
func (d *Dependency) Type() DependencyType { return d.typ }

// Runtime reports whether this is a runtime dependency (Gem::Dependency#runtime?).
func (d *Dependency) Runtime() bool { return d.typ == RuntimeType }

// SetPrerelease forces this dependency to be treated as a prerelease
// (Gem::Dependency#prerelease=).
func (d *Dependency) SetPrerelease(p bool) { d.prerelease = p }

// Prerelease reports whether the dependency requires a prerelease, either
// because it was forced or because its requirement is a prerelease
// (Gem::Dependency#prerelease?).
func (d *Dependency) Prerelease() bool { return d.prerelease || d.requirement.Prerelease() }

// LatestVersion reports whether the dependency simply asks for the latest
// version, i.e. has no real requirement (Gem::Dependency#latest_version?).
func (d *Dependency) LatestVersion() bool { return d.requirement.None() }

// Specific reports whether the dependency will not always match the latest
// version (Gem::Dependency#specific?).
func (d *Dependency) Specific() bool { return d.requirement.Specific() }

// String returns the dependency's display form (Gem::Dependency#to_s).
func (d *Dependency) String() string {
	if d.Type() != RuntimeType {
		return fmt.Sprintf("%s (%s, %s)", d.Name, d.requirement.String(), d.Type())
	}
	return fmt.Sprintf("%s (%s)", d.Name, d.requirement.String())
}

// Equal reports dependency equality (Gem::Dependency#==): same name, type and
// requirement.
func (d *Dependency) Equal(other *Dependency) bool {
	return d.Name == other.Name &&
		d.Type() == other.Type() &&
		d.requirement.Equal(other.requirement)
}

// Compare orders dependencies by name (Gem::Dependency#<=>).
func (d *Dependency) Compare(other *Dependency) int {
	return strings.Compare(d.Name, other.Name)
}

// Match reports whether this dependency matches a gem identified by name and
// version (Gem::Dependency#match?). A prerelease version is rejected unless
// allowPrerelease is set or this dependency is itself a prerelease.
func (d *Dependency) Match(name string, version *Version, allowPrerelease bool) bool {
	if d.Name != name {
		return false
	}
	if d.requirement.None() && !version.Prerelease() {
		return true
	}
	if version.Prerelease() && !allowPrerelease && !d.Prerelease() {
		return false
	}
	return d.requirement.SatisfiedBy(version)
}

// MatchesSpec reports whether the dependency matches a spec
// (Gem::Dependency#matches_spec?). Unlike Match, a prerelease spec version is
// accepted even when the dependency is not a prerelease dependency.
func (d *Dependency) MatchesSpec(spec *Specification) bool {
	if d.Name != spec.Name {
		return false
	}
	if d.requirement.None() {
		return true
	}
	return d.requirement.SatisfiedBy(spec.Version)
}

// Merge merges the requirements of other into this dependency, returning a new
// Dependency (Gem::Dependency#merge). The names must match.
func (d *Dependency) Merge(other *Dependency) (*Dependency, error) {
	if d.Name != other.Name {
		return nil, fmt.Errorf("%s and %s have different names", d, other)
	}
	def := DefaultRequirement()
	if other.requirement.Equal(def) {
		return &Dependency{Name: d.Name, requirement: d.requirement, typ: d.typ}, nil
	}
	if d.requirement.Equal(def) {
		return &Dependency{Name: d.Name, requirement: other.requirement, typ: d.typ}, nil
	}
	// Both operands are valid requirements, so re-parsing their AsList strings
	// always succeeds (MustRequirement is total here).
	merged := append(append([]string{}, d.requirement.AsList()...), other.requirement.AsList()...)
	return &Dependency{Name: d.Name, requirement: MustRequirement(merged...), typ: d.typ}, nil
}
