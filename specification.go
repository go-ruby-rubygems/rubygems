package rubygems

import (
	"fmt"
	"regexp"
)

// nameRE validates a gem name, mirroring the printable-character check in
// Gem::Specification#validate (the name must not be nil/empty and must contain
// only printable ASCII without leading/trailing whitespace).
var nameRE = regexp.MustCompile(`\A[a-zA-Z0-9](?:[a-zA-Z0-9_.-]*[a-zA-Z0-9])?\z`)

// Specification is a usable subset of Gem::Specification: the in-memory
// metadata model plus the pure-compute methods (version/dependency fields,
// satisfies_requirement?, a validate-lite). The full gemspec eval, the file
// manifest, signing, and gem install are host-side and out of scope.
type Specification struct {
	Name         string
	Version      *Version
	Summary      string
	dependencies []*Dependency
}

// NewSpecification constructs a Specification from a name and version string.
func NewSpecification(name, version string) (*Specification, error) {
	v, err := NewVersion(version)
	if err != nil {
		return nil, err
	}
	return &Specification{Name: name, Version: v}, nil
}

// AddRuntimeDependency adds a runtime dependency
// (Gem::Specification#add_runtime_dependency).
func (s *Specification) AddRuntimeDependency(name string, constraints ...string) error {
	return s.addDependency(name, RuntimeType, constraints...)
}

// AddDevelopmentDependency adds a development dependency
// (Gem::Specification#add_development_dependency).
func (s *Specification) AddDevelopmentDependency(name string, constraints ...string) error {
	return s.addDependency(name, DevelopmentType, constraints...)
}

func (s *Specification) addDependency(name string, typ DependencyType, constraints ...string) error {
	dep, err := NewDependency(name, typ, constraints...)
	if err != nil {
		return err
	}
	s.dependencies = append(s.dependencies, dep)
	return nil
}

// Dependencies returns all dependencies (Gem::Specification#dependencies).
func (s *Specification) Dependencies() []*Dependency { return s.dependencies }

// RuntimeDependencies returns only the runtime dependencies
// (Gem::Specification#runtime_dependencies).
func (s *Specification) RuntimeDependencies() []*Dependency {
	var out []*Dependency
	for _, d := range s.dependencies {
		if d.Runtime() {
			out = append(out, d)
		}
	}
	return out
}

// DevelopmentDependencies returns only the development dependencies
// (Gem::Specification#development_dependencies).
func (s *Specification) DevelopmentDependencies() []*Dependency {
	var out []*Dependency
	for _, d := range s.dependencies {
		if !d.Runtime() {
			out = append(out, d)
		}
	}
	return out
}

// SatisfiesRequirement reports whether this spec satisfies a dependency: the
// names match and the dependency's requirement is satisfied by this spec's
// version (Gem::Specification#satisfies_requirement?).
func (s *Specification) SatisfiesRequirement(dep *Dependency) bool {
	return s.Name == dep.Name && dep.requirement.SatisfiedBy(s.Version)
}

// FullName returns "name-version" (Gem::Specification#full_name).
func (s *Specification) FullName() string {
	return s.Name + "-" + s.Version.String()
}

// Validate performs the pure-compute subset of Gem::Specification#validate that
// does not touch the filesystem: it checks that the name is a well-formed gem
// name and that the version is present. It returns an error describing the
// first problem found. The full validate (file lists, licenses, metadata URIs,
// signing) is host-side.
func (s *Specification) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("missing value for attribute name")
	}
	if !nameRE.MatchString(s.Name) {
		return fmt.Errorf("invalid value for attribute name: %q", s.Name)
	}
	if s.Version == nil {
		return fmt.Errorf("missing value for attribute version")
	}
	return nil
}
