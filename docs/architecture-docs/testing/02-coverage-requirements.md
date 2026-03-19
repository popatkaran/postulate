# Coverage Requirements and CI Configuration

**Document Version:** 1.0  
**Last Updated:** 2025-01-01  
**Owner:** Architecture Review Board  
**Status:** Approved  
**ADR Reference:** ADR-0003

---

## Purpose

This document defines the minimum code coverage requirements, how coverage is measured, how the CI pipeline enforces coverage gates, and what a team must do when coverage falls below the minimum threshold.

---

## Coverage Minimum Requirements

| Scope | Line | Branch | Function |
|-------|------|--------|----------|
| Overall project | 90% | 90% | 90% |
| Plugin Registry | 95% | 95% | 95% |
| Plugin Lifecycle Manager | 95% | 95% | 95% |
| Core Event Bus | 95% | 95% | 95% |
| Dependency Injection Container | 95% | 95% | 95% |

### What is Counted

Coverage is measured across all production source code. The following are included in coverage measurement:

- All classes in the Domain Layer.
- All classes in the Application Layer.
- All classes in the Infrastructure Layer.
- All classes in the Delivery Layer.
- All Plugin Registry and lifecycle management classes.

### What is Excluded from Coverage Measurement

The following are excluded because they cannot be meaningfully covered by automated tests or are framework-generated:

- Generated code (ORM entity proxies, protobuf-generated classes, OpenAPI-generated client stubs).
- Composition Root / bootstrap code (application entry point that wires the DI container).
- Configuration data classes with no logic (pure data containers).
- Test code itself.

Exclusion of any additional source must be explicitly approved and documented inline in the source file with a comment explaining the reason.

---

## Measuring Coverage

Coverage is measured by executing the test suite with a coverage instrumentation tool appropriate to the implementation language. Coverage results must be produced in a standard report format (e.g., LCOV, Cobertura XML, or JaCoCo XML) that can be consumed by CI tooling and coverage dashboards.

The coverage tool must be configured to:
1. Instrument production source code only (not test code, not generated code).
2. Measure line, branch, and function coverage simultaneously.
3. Produce a per-file and per-module breakdown in addition to the aggregate.
4. Output results in a machine-readable format for CI gate enforcement.

---

## CI Pipeline Integration

### Pipeline Stages and Coverage Gate Position

```
Stage 1: Code Checkout and Dependency Resolution
    |
    v
Stage 2: Static Analysis (linting, type checking)
    |
    v
Stage 3: Unit Tests (Tier 1) with coverage instrumentation
    |
    v
Stage 4: Coverage Gate Check -- FAIL if below 90% overall or 95% for critical modules
    |
    v
Stage 5: Integration Tests (Tier 2)
    |
    v
Stage 6: Contract Tests (Tier 3) for any plugin with changes in this PR
    |
    v
Stage 7: Build and Package
    |
    v
Stage 8: E2E Tests (Tier 4) -- on merge to main and on release branches only
```

The Coverage Gate Check in Stage 4 is a hard gate. If the gate fails, stages 5 through 8 do not execute and the pull request cannot be merged.

### Gate Configuration

The CI pipeline must be configured with explicit threshold assertions. The following illustrates the configuration intent in a language-agnostic format:

```
// CI configuration pseudo-structure

coverage_gates:
  overall:
    line_coverage_minimum: 90
    branch_coverage_minimum: 90
    function_coverage_minimum: 90
    on_failure: FAIL_BUILD

  critical_modules:
    - module: "plugin/registry"
      line_coverage_minimum: 95
      branch_coverage_minimum: 95
      function_coverage_minimum: 95
      on_failure: FAIL_BUILD
    - module: "plugin/lifecycle"
      line_coverage_minimum: 95
      branch_coverage_minimum: 95
      function_coverage_minimum: 95
      on_failure: FAIL_BUILD
    - module: "core/eventbus"
      line_coverage_minimum: 95
      branch_coverage_minimum: 95
      function_coverage_minimum: 95
      on_failure: FAIL_BUILD
```

### Pull Request Coverage Delta Reporting

In addition to the gate check, the CI pipeline must post a coverage delta report as a comment on every pull request. This report must show:

- Overall coverage before and after the change.
- Files changed in the PR with their individual coverage percentages.
- Files whose coverage decreased, highlighted for reviewer attention.

This makes coverage regression visible in code review without requiring reviewers to check CI logs.

---

## Coverage Exceptions

In rare circumstances, a specific file or method may be excluded from coverage measurement. The following rules govern exceptions:

1. The exclusion must be documented inline in the source code at the point of exclusion, with a comment stating the reason.
2. The exclusion must be recorded in `COVERAGE_EXCEPTIONS.md` at the repository root with: the file path, the reason, the date, and the approving architect.
3. Exclusions are reviewed at the Architecture Review Board quarterly. Any exclusion without a current justification is removed.

Example inline exclusion comment (syntax is indicative; use the actual format required by the coverage tool in use):

```
// coverage:ignore-start
// Reason: This is the application entry point / Composition Root.
// It wires the DI container and cannot be meaningfully unit tested.
// The system is validated at the E2E test level.
method main():
    bootstrap()
// coverage:ignore-end
```

---

## When Coverage Falls Below the Threshold

If a pull request causes overall coverage to fall below 90%, the following process applies:

1. The CI pipeline fails and merge is blocked.
2. The author must add tests to bring coverage back above the threshold.
3. The author must not suppress coverage measurement or move code to excluded locations as a workaround. This is a disciplinary matter if discovered in code review.

If the coverage fall was caused by the deletion of code whose tests were also deleted (a legitimate net reduction), the author must document that the deleted code had no test coverage debt and that the overall project coverage is now more accurate.

---

## Coverage Reporting and Dashboards

Coverage reports are published as artefacts of every CI run. Historical coverage trends are visible in the project's CI dashboard.

Coverage trend data must be retained for a minimum of 90 days. Teams are encouraged to monitor the trend, not just the gate. A slow, persistent downward trend in coverage is a signal that test discipline is weakening, even if the gate has not yet been breached.

---

## References

- `ADR-0003` — Testing Standards and Coverage Requirements decision
- `testing/01-testing-standards.md` — Test structure, naming, and tier definitions
- `testing/03-testing-patterns.md` — How to write testable code and effective test doubles
