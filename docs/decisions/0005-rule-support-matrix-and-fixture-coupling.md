# 0005 — The rule support matrix and the coupling with the fixtures

- Status: **Accepted**
- Date: 2026-10-20

## Context

The specification requires every rule to publish, for each of the five
languages, its support level (`none`, `syntactic`, `resolved`), the method
it uses and its known limitations. It also requires the matrix to be
maintained next to the fixtures and to be **validated in CI**: a rule may
not declare support for a language without positive and negative fixtures.

The reason is visible in the exit criterion of Phase 1: the five languages
are recognized from the first milestone, but a rule may only activate in the
languages where it has proven coverage. "Do not mark a language as
compatible with a rule if it has only been superficially tested."

The risk this decision protects against is concrete: a rule that claims
support for a language it has never been tested on produces findings that
cannot be trusted, and a user has no way of knowing that from the output.

## Decision

**A rule may only declare support for a language that has both a positive
and a negative fixture. The support level never exceeds `syntactic` in this
milestone. The epistemic status of a finding is part of the contract of the
rule.**

The concrete rules:

1. **The matrix lives in the rule definition.** Every rule declares a
   `Languages` map from the language identifier to its support level, and a
   `Limitations` map with the known limits per language. Both are published
   by `codixai audit --rules` and serialized in the JSON catalog.
2. **The fixtures are the enforcement mechanism.** The corpus lives in
   `testdata/rules/<RULE-ID>/<language>/{positive,negative}/`. Each case
   holds a `<name>.diff` and a `<name>.expected.json` with
   `expected_rule_ids` and `expected_count`. A case may hold a `repo/`
   directory with the manifests of the analyzed project.
3. **CI validates the matrix against the corpus.**
   `TestSupportMatrixMatchesFixtures` in `internal/analysis/engine_test.go`
   fails when a rule declares support for a language that has no fixture, or
   when a fixture exists for a combination that the matrix declares as
   `none`. The CI workflow runs that test explicitly, in its own job.
4. **The support level never exceeds `syntactic` in this milestone.**
   `resolved` requires semantically resolved relations, and the analysis
   resolves none. The level is a property of the implementation, not a wish.
5. **The epistemic status is part of the contract of the rule.** Every
   finding carries an `epistemic_status`: `observed` when the finding was
   checked directly against the code or the diff, `inferred` when it comes
   from a heuristic, `unknown` when the tool could not determine the
   outcome. A heuristic signal is never published as a confirmed bug, and a
   rule that mixes the two states must document the difference in its
   limitations.
6. **The identifier and the version are part of the contract.** The
   identifier follows `CDX-<AREA>-<NNN>`, is never reused with another
   meaning, and the version is incremented when the behavior of the rule
   changes. `TestRuleIdentifiersFollowTheConvention` in
   `internal/analysis/rules/catalog_test.go` enforces the convention.

## Consequences

Positive:

- The support matrix is a truthful statement about the coverage, and it is
  checked instead of declared.
- A user reading the output can tell, per rule and per language, how deep
  the analysis went and what it could not check.
- The fixtures are the shared definition of "the rule works here", which
  keeps the corpus and the matrix from drifting apart.
- The epistemic status makes the difference between a fact and a signal
  explicit in every document.

Negative:

- Adding a language means adding fixtures for every rule that claims
  support for it, which is more work than declaring the support.
- A rule that would be useful in a language without fixtures stays
  `none` there, which means fewer findings and a smaller apparent value.
- The corpus has to be maintained: a change in the behavior of a rule
  requires a version bump and a fixture review.

## Alternatives considered

**Publish the matrix without validating it against the fixtures.**
Rejected. A matrix that nothing checks drifts, and a drifted matrix is
worse than no matrix, because the user trusts it.

**Validate only the presence of a positive fixture, and accept a language
without a negative case.**
Rejected. A negative case is what proves that the rule does not fire on
ordinary code. Without it, a rule that always fires would pass the
validation.

**Declare `resolved` for the rules that use `go/parser`.**
Rejected. `go/parser` provides a syntax tree, not a resolved relation. The
scope of the Go reference rule is the package directory of the changed
file, and identifiers declared in another package of the module are
reported as missing. The level stays `syntactic`.

**Let a rule declare its support level per case, in the fixture.**
Rejected. The support level is a property of the rule and of the language,
not of a test case. The fixture declares the expected result; the matrix
declares the coverage.

**Use a single fixture directory per rule, without the language level.**
Rejected. The language is the axis that matters: it is the axis the
specification uses to publish the support, and the axis on which the rules
differ.
