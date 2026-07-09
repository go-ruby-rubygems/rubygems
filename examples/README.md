# Examples

Runnable pure-Ruby usage of the `Gem::Version` / `Gem::Requirement` algebra, verified under the [rbgo](https://github.com/go-embedded-ruby/ruby) interpreter.

```sh
rbgo examples/rubygems_usage.rb
```

| File | Shows |
| --- | --- |
| `rubygems_usage.rb` | Semantic version comparison and sorting, prerelease detection, `Gem::Version.correct?`, trailing-zero equality, and `Gem::Requirement#satisfied_by?` for pessimistic (`~>`) and multi-constraint requirements |
