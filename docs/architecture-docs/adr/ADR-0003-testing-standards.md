# ADR-0003: Testing Standards and Minimum Coverage Requirements

**Status:** Accepted  
**Date:** 2025-01-01  
**Deciders:** Architecture Review Board  
**Ticket / RFC Reference:** RFC-003

---

## Context

A plugin-first, highly extensible platform carries compounded risk: a defect in the core runtime can cascade to all plugins; a defect in a plugin contract can silently corrupt results across all implementations. Without formalised testing standards, coverage tends to erode over time as deadlines create pressure to ship without tests.

Quality must be a structural property of the repository, not a per-sprint effort. This requires both a minimum quantitative bar (coverage percentage) and qualitative standards (what types of tests to write, and when).

## Decision Drivers

- Confidence in refactoring: tests must provide a safety net dense enough that any regression is caught before merge.
- The plugin lifecycle and Registry are critical path; they require the highest coverage.
- Tests must be executable in CI on every pull request with a clear pass/fail signal.
- Test code must be held to the same structural quality standards as production code.
- The 90% minimum is a floor, not a target; teams are encouraged to exceed it.

## Decision

**The minimum acceptable test coverage for this project is 90% across all measurable dimensions: line coverage, branch coverage, and function coverage. Coverage is measured in CI on every pull request. A pull request that reduces overall coverage below 90% will not be merged.**

Critical path modules — defined as Plugin Registry, Plugin Lifecycle Manager, Core Event Bus, and Dependency Injection Container — must maintain a minimum of 95% coverage individually.

---

## Test Classification

All tests in this project are classified into four tiers. Each tier has a defined scope, a defined speed budget, and a defined location in the CI pipeline.

### Tier 1 — Unit Tests

**Scope:** A single class or pure function in isolation. All collaborators are replaced by test doubles (mocks, stubs, or fakes).  
**Speed budget:** Each test must complete in under 50 milliseconds.  
**Location in CI:** Run on every commit push, before build.  
**Coverage contribution:** Primary contributor to the 90% floor.

Unit tests verify that a unit of logic does what its contract says, given controlled inputs. They do not test integration with other units.

### Tier 2 — Integration Tests

**Scope:** Two or more collaborating units, exercising real interactions between them, but with external I/O (databases, file systems, network) replaced by in-process fakes or test containers.  
**Speed budget:** Each test must complete in under 2 seconds.  
**Location in CI:** Run on every pull request, after unit tests pass.

### Tier 3 — Contract Tests

**Scope:** Verification that a plugin implementation correctly satisfies the Plugin Contract. Contract tests are written once against the interface and run against every implementation.  
**Speed budget:** Each test must complete in under 500 milliseconds.  
**Location in CI:** Run against every plugin on every pull request touching that plugin.

This is the primary mechanism for enforcing LSP compliance in the plugin system.

### Tier 4 — End-to-End Tests

**Scope:** Full system behaviour from an external entry point, with a real or near-real infrastructure stack.  
**Speed budget:** Suite must complete in under 15 minutes total.  
**Location in CI:** Run on merge to main branch and on release candidates.

---

## Test Quality Standards

### Naming Convention

Test names must express a complete behavioural assertion, not a method name.

```
// POOR — describes what is called, not what is expected
test_save_user()

// GOOD — describes the scenario and the expected outcome
test_given_valid_user_when_register_then_user_is_persisted()
test_given_duplicate_email_when_register_then_DuplicateEmailException_is_raised()
```

### Arrange-Act-Assert (AAA) Structure

Every test must follow the AAA pattern with a blank line separating each section.

```
// Pseudo-code
test "given an active plugin, when the plugin is stopped, then its status becomes STOPPED":

    // Arrange
    plugin = new ConcretePlugin()
    registry = new PluginRegistry()
    registry.register(plugin)
    registry.start(plugin.id)

    // Act
    registry.stop(plugin.id)

    // Assert
    assert registry.getStatus(plugin.id) == PluginStatus.STOPPED
```

### One Logical Assertion Per Test

A test must verify one and only one behaviour. Multiple unrelated assertions in a single test obscure the source of failure and reduce diagnostic value.

```
// VIOLATION — two independent behaviours in one test
test_registration_and_status():
    registry.register(plugin)
    assert registry.contains(plugin.id)          // behaviour 1
    assert registry.getStatus(plugin.id) == ACTIVE  // behaviour 2

// COMPLIANT — two focused tests
test_given_plugin_when_registered_then_registry_contains_it():
    registry.register(plugin)
    assert registry.contains(plugin.id)

test_given_plugin_when_registered_then_status_is_active():
    registry.register(plugin)
    assert registry.getStatus(plugin.id) == PluginStatus.ACTIVE
```

### Test Independence

Tests must not share mutable state. Each test must set up its own preconditions and must not depend on the execution order of other tests. Global state must be reset between tests.

### Test Doubles Policy

| Collaborator Type | Preferred Double Type | Rationale |
|---|---|---|
| Simple value objects | Real instance | No benefit to substituting |
| External I/O (DB, HTTP) | Fake (in-memory implementation) | More reliable than mocks for stateful interactions |
| Single-method callbacks | Stub | Minimal setup, focused |
| Collaborators with complex behaviour to verify | Mock | Verify interaction when state is not observable |

Prefer fakes and stubs over mocks. Mocks that assert on internal implementation details couple tests to implementation, making refactoring brittle.

---

## Consequences

### Positive

- The 90% floor creates a structural incentive to write tests during development rather than as a retrofit.
- Contract tests enforce LSP compliance for every plugin, providing a systematic quality gate.
- The tiered model keeps CI feedback loops fast: unit tests run in seconds.

### Negative

- Teams cannot ship a feature without meeting the coverage requirement, which adds short-term velocity cost.
- Achieving 90% branch coverage requires deliberate attention to edge cases and error paths.

### Neutral / Follow-up Actions

- Configure CI pipeline to measure and enforce coverage on every pull request.
- Document test double patterns and factory helpers in `testing/03-testing-patterns.md`.
- Create shared contract test base classes for the Plugin Contract.

## Compliance

- Test code must comply with SOLID principles (ADR-0002), in particular SRP: each test class tests one unit.
- All pseudo-code examples in architecture documents must have corresponding real test implementations before the feature is merged.

## References

- `testing/01-testing-standards.md` — Full testing standards reference
- `testing/02-coverage-requirements.md` — Coverage tooling and CI configuration guide
- `testing/03-testing-patterns.md` — Test patterns and double strategies
- `ADR-0002` — SOLID Principles Enforcement Policy
