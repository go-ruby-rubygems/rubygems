// Copyright (c) the go-ruby-rubygems/rubygems authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubygems

import (
	"reflect"
	"testing"
)

func TestNewVersionMalformed(t *testing.T) {
	for _, s := range []string{"junk", "1..2", "1.2 3", "1.2.", "-", "1.0-", "++"} {
		if _, err := NewVersion(s); err == nil {
			t.Errorf("NewVersion(%q): expected error", s)
		}
	}
	if got := MustVersion("").String(); got != "0" {
		t.Errorf("empty version => %q, want 0", got)
	}
	if got := MustVersion("  ").String(); got != "0" {
		t.Errorf("whitespace version => %q, want 0", got)
	}
	if got := MustVersion(" 1.0 ").String(); got != "1.0" {
		t.Errorf("trimmed version => %q, want 1.0", got)
	}
	if got := MustVersion("1-2").String(); got != "1.pre.2" {
		t.Errorf("dash version => %q, want 1.pre.2", got)
	}
}

func TestMustVersionPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("MustVersion did not panic on bad input")
		}
	}()
	MustVersion("garbage!!")
}

func TestCorrectVersion(t *testing.T) {
	if !CorrectVersion("1.2.3") {
		t.Error("1.2.3 should be correct")
	}
	if CorrectVersion("a.b.c") {
		t.Error("a.b.c should not be correct")
	}
	if !CorrectVersion("") {
		t.Error("empty should be correct")
	}
}

func TestCompareTable(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0", "1.0.0", 0},
		{"1.0.a", "1.0", -1},
		{"1.0", "1.0.a", 1},
		{"1.0.0.rc1", "1.0.0", -1},
		{"1.0.0.rc1", "1.0.0.rc.1", 0},
		{"1.0.a10", "1.0.a9", 1},
		{"2.0.b", "2.0.a", 1},
		{"1.2.3", "1.2.3.0.0", 0},
		{"1.0.0", "1.0.0.a", 1},
		{"0.9", "1.0.b1", -1},
		{"1.0.b1", "1.0.a.2", 1},
		{"1", "1.0.0.0.0.0.0.1", -1},
		{"3.10", "3.2", 1},
		{"1.0.0", "1.0.0", 0},
		{"2.0", "1.9.9", 1},
		// same-index type mismatch in the comparison loop: at index 1,
		// "1.a" has the string "a" while "1.2" has the number 2.
		{"1.a", "1.2", -1}, // String < Numeric
		{"1.2", "1.a", 1},  // Numeric > String
		// numeric tie-break at the same index (segCompare numeric path)
		{"1.3", "1.2", 1},
		// left side longer with a non-zero numeric tail (lhs tail loop, i++)
		{"1.0.0.0.1", "1.0", 1},
		// left side longer with only zero tail (lhs tail loop returns 0)
		{"1.0.0.0", "1", 0},
		// left side longer with a string tail
		{"1.0.0.a", "1.0.0.0.0", -1},
	}
	for _, c := range cases {
		got := MustVersion(c.a).Compare(MustVersion(c.b))
		if got != c.want {
			t.Errorf("Compare(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestEqualEqlHash(t *testing.T) {
	a := MustVersion("1.0")
	b := MustVersion("1.0.0")
	c := MustVersion("1")
	if !a.Equal(b) {
		t.Error("1.0 should == 1.0.0")
	}
	if a.Eql(b) {
		t.Error("1.0 should not eql? 1.0.0")
	}
	if !a.Eql(MustVersion("1.0")) {
		t.Error("1.0 should eql? 1.0")
	}
	if a.Eql(c) {
		t.Error("1.0 should not eql? 1")
	}
	// Equal versions share a HashKey.
	if a.HashKey() != b.HashKey() {
		t.Errorf("HashKey mismatch for equal versions: %q vs %q", a.HashKey(), b.HashKey())
	}
	// prerelease HashKey exercises the string branch.
	if MustVersion("1.0.a").HashKey() == a.HashKey() {
		t.Error("prerelease HashKey should differ")
	}
}

func TestSegmentsAndCanonical(t *testing.T) {
	v := MustVersion("1.0.0.rc1")
	if got, want := v.Segments(), []any{int64(1), int64(0), int64(0), "rc", int64(1)}; !reflect.DeepEqual(got, want) {
		t.Errorf("Segments=%v want %v", got, want)
	}
	if got, want := v.CanonicalSegments(), []any{int64(1), "rc", int64(1)}; !reflect.DeepEqual(got, want) {
		t.Errorf("CanonicalSegments=%v want %v", got, want)
	}
	// prerelease leading-zero strip: "0.a" -> canonical ["a"]
	if got, want := MustVersion("0.a").CanonicalSegments(), []any{"a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("0.a canonical=%v want %v", got, want)
	}
}

func TestPrerelease(t *testing.T) {
	if !MustVersion("1.0.a").Prerelease() {
		t.Error("1.0.a is prerelease")
	}
	if MustVersion("1.0.0").Prerelease() {
		t.Error("1.0.0 is not prerelease")
	}
}

func TestBumpReleaseApprox(t *testing.T) {
	cases := []struct {
		v, bump, release, approx string
	}{
		{"1.0.0", "1.1", "1.0.0", "~> 1.0"},
		{"5.3.1", "5.4", "5.3.1", "~> 5.3"},
		{"5.3.1.b.2", "5.4", "5.3.1", "~> 5.3.a"},
		{"1.0.0.beta.1", "1.1", "1.0.0", "~> 1.0.a"},
		{"1", "2", "1", "~> 1.0"},
		{"3.5.0", "3.6", "3.5.0", "~> 3.5"},
	}
	for _, c := range cases {
		if got := MustVersion(c.v).Bump().String(); got != c.bump {
			t.Errorf("Bump(%q)=%q want %q", c.v, got, c.bump)
		}
		if got := MustVersion(c.v).Release().String(); got != c.release {
			t.Errorf("Release(%q)=%q want %q", c.v, got, c.release)
		}
		if got := MustVersion(c.v).ApproximateRecommendation(); got != c.approx {
			t.Errorf("Approx(%q)=%q want %q", c.v, got, c.approx)
		}
	}
	// Release of a non-prerelease returns itself (identity branch).
	v := MustVersion("1.2.3")
	if v.Release() != v {
		t.Error("Release of non-prerelease should be identity")
	}
}
