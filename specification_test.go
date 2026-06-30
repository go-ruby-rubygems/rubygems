// Copyright (c) the go-ruby-rubygems/rubygems authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubygems

import "testing"

func TestNewSpecification(t *testing.T) {
	s, err := NewSpecification("mygem", "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "mygem" || s.Version.String() != "1.2.3" {
		t.Errorf("spec fields wrong")
	}
	if s.FullName() != "mygem-1.2.3" {
		t.Errorf("full_name => %q", s.FullName())
	}
	if _, err := NewSpecification("x", "bad!!"); err == nil {
		t.Error("bad version should error")
	}
}

func TestSpecDependencies(t *testing.T) {
	s, _ := NewSpecification("mygem", "1.0.0")
	if err := s.AddRuntimeDependency("rake", "~> 13.0"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddDevelopmentDependency("rspec", ">= 3.0"); err != nil {
		t.Fatal(err)
	}
	if len(s.Dependencies()) != 2 {
		t.Errorf("expected 2 deps, got %d", len(s.Dependencies()))
	}
	if len(s.RuntimeDependencies()) != 1 || s.RuntimeDependencies()[0].Name != "rake" {
		t.Error("runtime deps wrong")
	}
	if len(s.DevelopmentDependencies()) != 1 || s.DevelopmentDependencies()[0].Name != "rspec" {
		t.Error("dev deps wrong")
	}
	// error path: bad constraint
	if err := s.AddRuntimeDependency("x", "totally bad"); err == nil {
		t.Error("bad runtime dep should error")
	}
	if err := s.AddDevelopmentDependency("x", "totally bad"); err == nil {
		t.Error("bad dev dep should error")
	}
}

func TestSatisfiesRequirement(t *testing.T) {
	s, _ := NewSpecification("mygem", "1.5.0")
	d := MustDependency("mygem", RuntimeType, "~> 1.0")
	if !s.SatisfiesRequirement(d) {
		t.Error("1.5.0 satisfies ~> 1.0")
	}
	d2 := MustDependency("mygem", RuntimeType, "~> 2.0")
	if s.SatisfiesRequirement(d2) {
		t.Error("1.5.0 does not satisfy ~> 2.0")
	}
	d3 := MustDependency("other", RuntimeType, "~> 1.0")
	if s.SatisfiesRequirement(d3) {
		t.Error("name mismatch")
	}
}

func TestSpecValidate(t *testing.T) {
	s, _ := NewSpecification("good-gem_1.0", "1.0.0")
	if err := s.Validate(); err != nil {
		t.Errorf("valid spec errored: %v", err)
	}
	// empty name
	bad := &Specification{Name: "", Version: MustVersion("1.0")}
	if err := bad.Validate(); err == nil {
		t.Error("empty name should be invalid")
	}
	// invalid name chars
	bad2 := &Specification{Name: "bad name!", Version: MustVersion("1.0")}
	if err := bad2.Validate(); err == nil {
		t.Error("invalid name should error")
	}
	// missing version
	bad3 := &Specification{Name: "ok"}
	if err := bad3.Validate(); err == nil {
		t.Error("missing version should error")
	}
}
