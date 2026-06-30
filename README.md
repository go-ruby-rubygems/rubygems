<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-rubygems/brand/main/social/go-ruby-rubygems-rubygems.png" alt="go-ruby-rubygems/rubygems" width="720"></p>

# rubygems — go-ruby-rubygems

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-rubygems.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the pure-compute core of Ruby's
[RubyGems](https://github.com/rubygems/rubygems)** — `Gem::Version`,
`Gem::Requirement`, `Gem::Dependency` and a usable subset of
`Gem::Specification`. These are the comparable, in-memory version algebra that
the entire gem ecosystem is built on. The library is **byte-for-byte faithful**
to MRI's bundled RubyGems (target 3.6.x; the algorithms are unchanged through
RubyGems 4.x) on the two methods that matter most — `Gem::Version#<=>` and
`Gem::Requirement#satisfied_by?` (including the pessimistic `~>` bound math) —
validated against the `ruby` binary on every supported platform.

It is a sibling of [go-ruby-regexp](https://github.com/go-ruby-regexp/regexp)
(the Onigmo engine), [go-ruby-yaml](https://github.com/go-ruby-yaml/yaml) (the
Psych port) and [go-ruby-erb](https://github.com/go-ruby-erb/erb) (the ERB
compiler), and the version-algebra foundation that
[rbgo](https://github.com/go-embedded-ruby/ruby) and a future
`go-ruby-bundler` build their dependency resolution on — keeping the whole
stack **CGO=0**.

> **What it is — and isn't.** The version algebra — parsing a version string,
> comparing two versions, deciding whether a version satisfies a requirement —
> is fully deterministic and needs **no interpreter, no network, no
> filesystem**, so it lives here as pure Go. Fetching the gem index,
> downloading and installing `.gem` files, evaluating a real gemspec (arbitrary
> Ruby) and `require`-time activation are **host-side** concerns and are out of
> scope. See [`doc.go`](doc.go) for the precise boundary.

## Features

### `Gem::Version`

- **Parsing** of version strings (`"1.2.3"`, `"1.0.0.beta.1"`), with the
  `"-" -> ".pre."` rewrite and empty-string-as-`0` rule.
- **Prerelease detection** — a version is a prerelease iff it contains a letter.
- **The exact `<=>` algorithm**: split into numeric / letter segments,
  canonicalize (strip trailing zeros; strip leading zeros before the first
  prerelease letter), then compare segment-by-segment with `prerelease < release`
  and `String < Numeric` at each position. Matches RubyGems on the full matrix,
  including `"1.0" == "1.0.0"`, `"1.0.a" < "1.0"`, and `"1.0.0.rc1" < "1.0.0"`.
- `Bump`, `Release`, `Prerelease`, `CanonicalSegments`, `Segments`,
  `ApproximateRecommendation`, `String`, `Equal` (`==`), `Eql` (`eql?`),
  `HashKey` (a map-safe `hash` surrogate).

### `Gem::Requirement`

- **Constraint parsing** — `"~> 1.2"`, `">= 1.0"`, `"!= 2.0"`, bare versions
  (default `=`), and multiple constraints; duplicates removed, empty set ->
  `">= 0"`.
- **`SatisfiedBy`** with all operators (`=`, `!=`, `>`, `>=`, `<`, `<=`, `~>`),
  including the pessimistic `~>` bound math: `~> 1.2` is `>= 1.2, < 2.0`;
  `~> 1.2.3` is `>= 1.2.3, < 1.3.0`; `~> 1` is `>= 1.0, < 2.0`.
- `None`, `Exact`, `Specific`, `Prerelease`, `AsList`, `String`, and `Equal`
  (`==`) with the tilde-precision refinement (`~> 1.2 != ~> 1.2.0` while
  `>= 1.2 == >= 1.2.0`).

### `Gem::Dependency`

- Name + `Requirement` + type (`runtime` / `development`), `Match`,
  `MatchesSpec`, `Merge`, `Prerelease`, `LatestVersion`, `Specific`, `Compare`
  (by name), `Equal` and `String`.

### `Gem::Specification` (subset)

- The metadata model (name, version, runtime/development dependency fields),
  `SatisfiesRequirement`, `FullName`, and a filesystem-free `Validate`-lite.
  The full gemspec eval, file manifest, signing and gem install are host-side.

## Usage

```go
import "github.com/go-ruby-rubygems/rubygems"

v1 := rubygems.MustVersion("1.0.0.rc1")
v2 := rubygems.MustVersion("1.0.0")
v1.Compare(v2)          // -1  (a prerelease sorts below its release)

req := rubygems.MustRequirement("~> 1.2")
req.SatisfiedBy(rubygems.MustVersion("1.9.9")) // true
req.SatisfiedBy(rubygems.MustVersion("2.0.0")) // false
```

## Tests & coverage

`go test ./...` runs a deterministic, Ruby-free suite (which alone holds
coverage at **100%**, error branches included) plus a differential **MRI
oracle**: it shells out to the real `ruby` binary and checks `Version.Compare`
against `Gem::Version#<=>` and `Requirement.SatisfiedBy` against
`Gem::Requirement#satisfied_by?` over a broad version × version × operator
matrix. The oracle skips itself where `ruby` is absent (the Windows lane and the
qemu cross-arch lanes), so the deterministic suite keeps the gate green there.

CI builds and tests on **ubuntu / macos / windows** and on all **six 64-bit Go
targets** (amd64, arm64 native; riscv64, loong64, ppc64le, s390x under
qemu-user). The library is CGO=0 and has no third-party dependencies.

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright (c) 2026, the
go-ruby-rubygems/rubygems authors.
