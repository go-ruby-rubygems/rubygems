// Copyright (c) the go-ruby-rubygems/rubygems authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubygems

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// rubyBin locates a usable `ruby` once. The oracle tests skip themselves when
// it is absent (the qemu cross-arch lanes and the Windows lane), so the
// deterministic suite alone drives the 100% gate there.
func rubyBin(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping MRI RubyGems oracle")
	}
	return path
}

// rubyEval runs a Ruby script through `ruby -e` and returns stdout. The script
// calls $stdout.binmode first so Windows text-mode never rewrites "\n" -> "\r\n"
// (the go-ruby-erb lesson); the preamble here injects that for every script.
func rubyEval(t *testing.T, bin, script string) string {
	t.Helper()
	cmd := exec.Command(bin, "-rrubygems", "-e", "$stdout.binmode\n"+script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\nscript:\n%s\noutput:\n%s", err, script, out)
	}
	return string(out)
}

// rubyArray renders a Go []string as a Ruby array literal.
func rubyArray(ss []string) string {
	q := make([]string, len(ss))
	for i, s := range ss {
		q[i] = strconv.Quote(s)
	}
	return "[" + strings.Join(q, ", ") + "]"
}

// oracleVersions is the version corpus shared by the oracle tests. It spans
// releases, prereleases (letter segments), dash-tails ("-" -> ".pre."),
// trailing-zero canonicalization, and the "1.0 == 1.0.0" / "1.0.a < 1.0" /
// "1.0.0.rc1 vs 1.0.0" cases called out in the spec.
var oracleVersions = []string{
	"0", "0.0", "1", "1.0", "1.0.0", "1.0.0.0", "0.9", "1.0.b1", "1.0.a.2",
	"1.0.a", "1.0.0.rc1", "1.0.0.rc.1", "1.0.a10", "1.0.a9", "2.0.b", "2.0.a",
	"1.2.3", "1.2.3.0.0", "1.0.0.a", "1.0.0.beta.1", "10.0", "2.10", "2.2",
	"1.0.0-rc1", "1.2.3-foo", "0.0.1.alpha", "3.10", "3.2", "1.99", "1.0.0.0.1",
	"0.a", "5.3.1", "5.3.1.b.2", "1.0.0.pre", "1.0.0.pre.1", "1.0.0.pre.2",
	"1.a", "1.2", "1.3",
}

// TestOracleVersionCompare cross-checks Version.Compare against MRI's
// Gem::Version#<=> over the full N*N matrix of the corpus. This is the headline
// parity claim: byte-for-byte agreement with RubyGems on the comparison.
func TestOracleVersionCompare(t *testing.T) {
	bin := rubyBin(t)
	script := "vs = " + rubyArray(oracleVersions) + "\n" + `
vs.each do |a|
  va = Gem::Version.new(a)
  vs.each do |b|
    puts "#{va <=> Gem::Version.new(b)}"
  end
end
`
	out := rubyEval(t, bin, script)
	want := strings.Split(strings.TrimRight(out, "\n"), "\n")
	idx := 0
	for _, a := range oracleVersions {
		for _, b := range oracleVersions {
			got := MustVersion(a).Compare(MustVersion(b))
			w, _ := strconv.Atoi(want[idx])
			if got != w {
				t.Errorf("Compare(%q,%q)=%d, MRI=%d", a, b, got, w)
			}
			idx++
		}
	}
}

// TestOracleVersionMethods cross-checks bump / release / prerelease? /
// approximate_recommendation against MRI for each corpus version.
func TestOracleVersionMethods(t *testing.T) {
	bin := rubyBin(t)
	script := "vs = " + rubyArray(oracleVersions) + "\n" + `
vs.each do |a|
  va = Gem::Version.new(a)
  puts "#{va.bump}\t#{va.release}\t#{va.prerelease?}\t#{va.approximate_recommendation}"
end
`
	out := rubyEval(t, bin, script)
	sc := bufio.NewScanner(strings.NewReader(out))
	for _, a := range oracleVersions {
		if !sc.Scan() {
			t.Fatalf("ran out of MRI output at %q", a)
		}
		f := strings.Split(sc.Text(), "\t")
		v := MustVersion(a)
		if got := v.Bump().String(); got != f[0] {
			t.Errorf("Bump(%q)=%q MRI=%q", a, got, f[0])
		}
		if got := v.Release().String(); got != f[1] {
			t.Errorf("Release(%q)=%q MRI=%q", a, got, f[1])
		}
		if got := strconv.FormatBool(v.Prerelease()); got != f[2] {
			t.Errorf("Prerelease(%q)=%s MRI=%s", a, got, f[2])
		}
		if got := v.ApproximateRecommendation(); got != f[3] {
			t.Errorf("Approx(%q)=%q MRI=%q", a, got, f[3])
		}
	}
}

// TestOracleSatisfiedBy cross-checks Requirement.SatisfiedBy against MRI's
// Gem::Requirement#satisfied_by? for every (operator, requirement-version,
// candidate-version) triple, including the pessimistic "~>" bound math.
func TestOracleSatisfiedBy(t *testing.T) {
	bin := rubyBin(t)
	opsList := []string{"=", "!=", ">", "<", ">=", "<=", "~>"}
	script := "vs = " + rubyArray(oracleVersions) + "\nops = " + rubyArray(opsList) + "\n" + `
ops.each do |op|
  vs.each do |r|
    req = Gem::Requirement.new("#{op} #{r}")
    vs.each do |v|
      puts "#{req.satisfied_by?(Gem::Version.new(v))}"
    end
  end
end
`
	out := rubyEval(t, bin, script)
	want := strings.Split(strings.TrimRight(out, "\n"), "\n")
	idx := 0
	for _, op := range opsList {
		for _, r := range oracleVersions {
			req := MustRequirement(op + " " + r)
			for _, v := range oracleVersions {
				got := strconv.FormatBool(req.SatisfiedBy(MustVersion(v)))
				if got != want[idx] {
					t.Errorf("(%s %s) satisfied_by? %s = %s, MRI=%s", op, r, v, got, want[idx])
				}
				idx++
			}
		}
	}
}

// TestOracleMultiConstraint checks multi-constraint requirements (the most
// common real-world shape, e.g. ">= 1.0, < 2.0") against MRI.
func TestOracleMultiConstraint(t *testing.T) {
	bin := rubyBin(t)
	script := `
req = Gem::Requirement.new(">= 1.0", "< 2.0", "!= 1.5")
[%w[0.9 1.0 1.4 1.5 1.9 2.0 2.5]].flatten.each do |v|
  puts "#{req.satisfied_by?(Gem::Version.new(v))}"
end
`
	out := rubyEval(t, bin, script)
	want := strings.Split(strings.TrimRight(out, "\n"), "\n")
	req := MustRequirement(">= 1.0", "< 2.0", "!= 1.5")
	for i, v := range []string{"0.9", "1.0", "1.4", "1.5", "1.9", "2.0", "2.5"} {
		got := strconv.FormatBool(req.SatisfiedBy(MustVersion(v)))
		if got != want[i] {
			t.Errorf("multi satisfied_by? %s = %s, MRI=%s", v, got, want[i])
		}
	}
}
