package rubygems

// The pure-compute / host-side boundary.
//
// This package re-implements only the parts of RubyGems that are pure,
// deterministic, in-memory computation — the version algebra that the entire
// gem ecosystem is built on. It is byte-for-byte faithful to MRI's bundled
// RubyGems (target 3.6.x; validated against the algorithms unchanged through
// RubyGems 4.x) on the two methods that matter most:
//
//   - Version.Compare        == Gem::Version#<=>
//   - Requirement.SatisfiedBy == Gem::Requirement#satisfied_by? (incl. "~>")
//
// IN SCOPE (pure compute):
//
//   - Gem::Version:      parsing, prerelease detection, the full segment
//                        comparison + canonicalization, Bump, Release,
//                        CanonicalSegments, ApproximateRecommendation,
//                        String/Equal/Eql/HashKey.
//   - Gem::Requirement:  constraint parsing ("~>", ">=", "!=", multiple),
//                        SatisfiedBy with the pessimistic "~>" bound math, the
//                        =/!=/>/>=/</<= operators, None/Exact/Specific/
//                        Prerelease/Equal.
//   - Gem::Dependency:   name + requirement + type, Match/MatchesSpec/Merge.
//   - Gem::Specification: the metadata model (name, version, dependency
//                        fields), SatisfiesRequirement, a validate-lite.
//
// OUT OF SCOPE (host-side):
//
//   - Fetching the gem index from a remote source (HTTP / compact index).
//   - Downloading, unpacking and installing .gem files.
//   - Evaluating a real gemspec (which is arbitrary Ruby code) — that is the
//     job of a Ruby evaluator such as rbgo, not this library.
//   - require-time activation / load-path manipulation.
//   - The filesystem-touching parts of Gem::Specification#validate (file
//     manifests, license files, signing, metadata URIs).
//
// Downstream, rbgo (the pure-Go Ruby) and go-ruby-bundler build their
// dependency-resolution and version-matching logic on top of these types,
// keeping the whole stack CGO=0.
