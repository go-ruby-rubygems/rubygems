# frozen_string_literal: true

# RubyGems version algebra: Gem::Version and Gem::Requirement.
# The Gem module is preloaded (as in MRI), so no require is needed.

# Parse and compare versions. Comparison is semantic, not lexical:
# "1.10.0" is greater than "1.2.3".
puts Gem::Version.new("1.2.3") < Gem::Version.new("1.10.0") # => true

# Sorting uses the same ordering.
versions = ["1.10.0", "1.2.3", "1.2.0"].map { |s| Gem::Version.new(s) }
puts versions.sort.map(&:to_s).inspect # => ["1.2.0", "1.2.3", "1.10.0"]

# Prerelease versions (any version containing a letter) sort before releases.
puts Gem::Version.new("2.0.0.rc1").prerelease? # => true

# Validate a version string before constructing one.
puts Gem::Version.correct?("1.2.3") # => true

# "1.0" and "1.0.0" are equal (trailing zeros are canonicalized away).
puts Gem::Version.new("1.0") == Gem::Version.new("1.0.0") # => true

# Requirements express constraints; satisfied_by? tests a version against them.
pessimistic = Gem::Requirement.new("~> 1.2.3")
puts pessimistic.satisfied_by?(Gem::Version.new("1.2.9")) # => true  (>= 1.2.3, < 1.3.0)
puts pessimistic.satisfied_by?(Gem::Version.new("1.3.0")) # => false

# Multiple constraints combine with logical AND.
range = Gem::Requirement.new(">= 1.0", "< 2.0")
puts range.to_s                                      # => ">= 1.0, < 2.0"
puts range.satisfied_by?(Gem::Version.new("1.5"))    # => true
