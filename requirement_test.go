// Copyright (c) the go-ruby-rubygems/rubygems authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubygems

import (
	"reflect"
	"testing"
)

func TestParseConstraint(t *testing.T) {
	op, v, err := ParseConstraint("~> 1.2")
	if err != nil || op != "~>" || v.String() != "1.2" {
		t.Errorf("parse ~> 1.2 => %q %v %v", op, v, err)
	}
	// bare version defaults to "="
	op, v, _ = ParseConstraint("1.0")
	if op != "=" || v.String() != "1.0" {
		t.Errorf("bare version => %q %v", op, v)
	}
	if _, _, err := ParseConstraint("?? 1.0"); err == nil {
		t.Error("expected error for bad operator")
	}
	if _, _, err := ParseConstraint("garbage"); err == nil {
		t.Error("expected error for garbage")
	}
}

func TestNewRequirementDefaults(t *testing.T) {
	r := MustRequirement()
	if r.String() != ">= 0" {
		t.Errorf("empty requirement => %q want >= 0", r.String())
	}
	if !r.None() {
		t.Error("empty requirement should be None")
	}
	// An empty-string constraint is ill-formed (matches MRI), not the default.
	if _, err := NewRequirement(""); err == nil {
		t.Error("empty-string constraint should error")
	}
	if _, err := NewRequirement("totally bad"); err == nil {
		t.Error("expected error for bad constraint")
	}
	// dedup: same constraint twice
	r3 := MustRequirement(">= 1.0", ">= 1.0")
	if len(r3.requirements) != 1 {
		t.Errorf("dedup failed: %d constraints", len(r3.requirements))
	}
}

func TestMustRequirementPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("MustRequirement did not panic")
		}
	}()
	MustRequirement("nope nope")
}

func TestSatisfiedBy(t *testing.T) {
	cases := []struct {
		req, ver string
		want     bool
	}{
		{"~> 1.2", "1.2.0", true},
		{"~> 1.2", "1.9.9", true},
		{"~> 1.2", "2.0.0", false},
		{"~> 1.2.3", "1.2.5", true},
		{"~> 1.2.3", "1.3.0", false},
		{"~> 1", "1.5", true},
		{"~> 1", "2.0", false},
		{">= 1.0", "1.0", true},
		{">= 1.0", "0.9", false},
		{"> 1.0", "1.0", false},
		{"> 1.0", "1.1", true},
		{"< 2.0", "1.9", true},
		{"< 2.0", "2.0", false},
		{"<= 2.0", "2.0", true},
		{"<= 2.0", "2.1", false},
		{"= 1.0", "1.0.0", true},
		{"= 1.0", "1.1", false},
		{"!= 2.0", "2.0", false},
		{"!= 2.0", "2.1", true},
	}
	for _, c := range cases {
		got := MustRequirement(c.req).SatisfiedBy(MustVersion(c.ver))
		if got != c.want {
			t.Errorf("%q sat %q => %v want %v", c.req, c.ver, got, c.want)
		}
	}
}

func TestMultiConstraint(t *testing.T) {
	r := MustRequirement(">= 1.0", "< 2.0")
	if !r.SatisfiedBy(MustVersion("1.5")) {
		t.Error("1.5 should satisfy >=1.0,<2.0")
	}
	if r.SatisfiedBy(MustVersion("2.5")) {
		t.Error("2.5 should not satisfy")
	}
	if r.String() != ">= 1.0, < 2.0" {
		t.Errorf("to_s => %q", r.String())
	}
	if !reflect.DeepEqual(r.AsList(), []string{">= 1.0", "< 2.0"}) {
		t.Errorf("as_list => %v", r.AsList())
	}
}

func TestRequirementPredicates(t *testing.T) {
	if MustRequirement("~> 1.2").None() {
		t.Error("~> 1.2 is not None")
	}
	if MustRequirement(">= 1.0", "< 2.0").None() {
		t.Error("multi is not None")
	}
	if !MustRequirement("= 1.0").Exact() {
		t.Error("= 1.0 is Exact")
	}
	if MustRequirement(">= 1.0").Exact() {
		t.Error(">= 1.0 is not Exact")
	}
	if MustRequirement(">= 1.0", "< 2.0").Exact() {
		t.Error("multi is not Exact")
	}
	if !MustRequirement(">= 1.0", "< 2.0").Specific() {
		t.Error("multi is Specific")
	}
	if !MustRequirement("= 1.0").Specific() {
		t.Error("= 1.0 is Specific")
	}
	if MustRequirement(">= 1.0").Specific() {
		t.Error(">= 1.0 is not Specific")
	}
	if MustRequirement("> 1.0").Specific() {
		t.Error("> 1.0 is not Specific")
	}
	if !MustRequirement("~> 1.0.a").Prerelease() {
		t.Error("~> 1.0.a is Prerelease")
	}
	if MustRequirement(">= 1.0").Prerelease() {
		t.Error(">= 1.0 is not Prerelease")
	}
}

func TestRequirementEqual(t *testing.T) {
	if !MustRequirement(">= 1.0").Equal(MustRequirement(">= 1.0")) {
		t.Error(">= 1.0 == >= 1.0")
	}
	if MustRequirement(">= 1.0").Equal(MustRequirement(">= 2.0")) {
		t.Error(">= 1.0 != >= 2.0")
	}
	// different length
	if MustRequirement(">= 1.0").Equal(MustRequirement(">= 1.0", "< 2.0")) {
		t.Error("different lengths not equal")
	}
	// tilde same precision
	if !MustRequirement("~> 1.2").Equal(MustRequirement("~> 1.2")) {
		t.Error("~> 1.2 == ~> 1.2")
	}
	// tilde different precision: ~> 1.2 vs ~> 1.2.0 have same sorted list? No,
	// as_list differs ("~> 1.2" vs "~> 1.2.0") so caught earlier — exercise the
	// tilde eql branch with an order difference instead.
	a := MustRequirement("~> 1.2", ">= 1.0")
	b := MustRequirement(">= 1.0", "~> 1.2")
	if !a.Equal(b) {
		t.Error("order-independent tilde equality failed")
	}
	// tilde precision refinement: "~> 1.2" and "~> 1.2.0" compare equal by
	// value (base check passes) but the stricter tilde eql? distinguishes
	// precision, so they are NOT == (matches MRI).
	if MustRequirement("~> 1.2").Equal(MustRequirement("~> 1.2.0")) {
		t.Error("~> 1.2 should not equal ~> 1.2.0 (precision)")
	}
	// By contrast, non-tilde operators DO treat 1.2 == 1.2.0 (value equality).
	if !MustRequirement(">= 1.2").Equal(MustRequirement(">= 1.2.0")) {
		t.Error(">= 1.2 should equal >= 1.2.0")
	}
}

func TestDefaultRequirements(t *testing.T) {
	if DefaultRequirement().String() != ">= 0" {
		t.Error("default")
	}
	if DefaultPrereleaseRequirement().String() != ">= 0.a" {
		t.Error("default prerelease")
	}
}
