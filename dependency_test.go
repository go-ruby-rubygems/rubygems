// Copyright (c) the go-ruby-rubygems/rubygems authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubygems

import "testing"

func TestNewDependency(t *testing.T) {
	d := MustDependency("rake", RuntimeType, "~> 13.0")
	if d.Name != "rake" || d.Type() != RuntimeType {
		t.Errorf("dependency fields wrong: %v", d)
	}
	if d.String() != "rake (~> 13.0)" {
		t.Errorf("to_s => %q", d.String())
	}
	// development type renders the type in to_s.
	dev := MustDependency("rspec", DevelopmentType, ">= 3.0")
	if dev.String() != "rspec (>= 3.0, development)" {
		t.Errorf("dev to_s => %q", dev.String())
	}
	if dev.Runtime() {
		t.Error("dev is not runtime")
	}
	// empty type defaults to runtime.
	def := MustDependency("a", "")
	if def.Type() != RuntimeType || !def.Runtime() {
		t.Error("empty type should default runtime")
	}
}

func TestDependencyErrors(t *testing.T) {
	if _, err := NewDependency("", RuntimeType); err == nil {
		t.Error("empty name should error")
	}
	if _, err := NewDependency("x", "weird"); err == nil {
		t.Error("bad type should error")
	}
	if _, err := NewDependency("x", RuntimeType, "totally bad"); err == nil {
		t.Error("bad constraint should error")
	}
}

func TestMustDependencyPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	MustDependency("", RuntimeType)
}

func TestDependencyPredicates(t *testing.T) {
	d := MustDependency("g", RuntimeType, "~> 1.0.a")
	if !d.Prerelease() {
		t.Error("~> 1.0.a dependency is prerelease")
	}
	d2 := MustDependency("g", RuntimeType)
	if !d2.LatestVersion() {
		t.Error("no-constraint dependency is latest_version")
	}
	if d2.Prerelease() {
		t.Error("default dependency not prerelease")
	}
	d2.SetPrerelease(true)
	if !d2.Prerelease() {
		t.Error("forced prerelease")
	}
	if !d.Specific() {
		t.Error("~> is specific")
	}
	if d.Compare(MustDependency("h", RuntimeType)) >= 0 {
		t.Error("g < h by name")
	}
	if d.Requirement().String() != "~> 1.0.a" {
		t.Errorf("requirement => %q", d.Requirement().String())
	}
}

func TestDependencyEqual(t *testing.T) {
	a := MustDependency("g", RuntimeType, ">= 1.0")
	b := MustDependency("g", RuntimeType, ">= 1.0")
	if !a.Equal(b) {
		t.Error("equal deps")
	}
	if a.Equal(MustDependency("h", RuntimeType, ">= 1.0")) {
		t.Error("different name")
	}
	if a.Equal(MustDependency("g", DevelopmentType, ">= 1.0")) {
		t.Error("different type")
	}
	if a.Equal(MustDependency("g", RuntimeType, ">= 2.0")) {
		t.Error("different requirement")
	}
}

func TestDependencyMatch(t *testing.T) {
	d := MustDependency("g", RuntimeType, "~> 1.0")
	if !d.Match("g", MustVersion("1.5"), false) {
		t.Error("1.5 matches ~> 1.0")
	}
	if d.Match("other", MustVersion("1.5"), false) {
		t.Error("name mismatch")
	}
	// none + non-prerelease => true
	none := MustDependency("g", RuntimeType)
	if !none.Match("g", MustVersion("1.5"), false) {
		t.Error("none matches release")
	}
	// none + prerelease, not allowed, not prerelease dep => false
	if none.Match("g", MustVersion("1.5.a"), false) {
		t.Error("prerelease rejected without allow")
	}
	// allowPrerelease overrides
	if !none.Match("g", MustVersion("1.5.a"), true) {
		t.Error("allowPrerelease should match")
	}
	// prerelease dependency accepts prerelease
	pre := MustDependency("g", RuntimeType, "~> 1.0.a")
	if !pre.Match("g", MustVersion("1.0.b"), false) {
		t.Error("prerelease dep matches prerelease version")
	}
}

func TestDependencyMatchesSpecAndMerge(t *testing.T) {
	spec, _ := NewSpecification("g", "1.5.0")
	d := MustDependency("g", RuntimeType, "~> 1.0")
	if !d.MatchesSpec(spec) {
		t.Error("spec matches")
	}
	if d.MatchesSpec(mustSpec(t, "other", "1.5.0")) {
		t.Error("name mismatch spec")
	}
	none := MustDependency("g", RuntimeType)
	if !none.MatchesSpec(spec) {
		t.Error("none matches any spec")
	}

	// Merge
	a := MustDependency("g", RuntimeType, ">= 1.0")
	b := MustDependency("g", RuntimeType, "< 2.0")
	m, err := a.Merge(b)
	if err != nil {
		t.Fatal(err)
	}
	if m.Requirement().String() != ">= 1.0, < 2.0" {
		t.Errorf("merged => %q", m.Requirement().String())
	}
	// merge with default on other => keep self
	def := MustDependency("g", RuntimeType)
	m2, _ := a.Merge(def)
	if m2.Requirement().String() != ">= 1.0" {
		t.Errorf("merge with default other => %q", m2.Requirement().String())
	}
	// merge with default on self => take other
	m3, _ := def.Merge(a)
	if m3.Requirement().String() != ">= 1.0" {
		t.Errorf("merge default self => %q", m3.Requirement().String())
	}
	// different names error
	if _, err := a.Merge(MustDependency("h", RuntimeType)); err == nil {
		t.Error("merge different names should error")
	}
}

func mustSpec(t *testing.T, name, ver string) *Specification {
	t.Helper()
	s, err := NewSpecification(name, ver)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
