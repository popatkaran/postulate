# Contributing Guide and Code Review Checklist

**Document Version:** 1.0  
**Last Updated:** 2025-01-01  
**Owner:** Architecture Review Board  
**Status:** Approved

---

## Purpose

This document defines the contribution process for this project: how to structure a pull request, what must be present before a pull request is opened, and the complete code review checklist that both authors and reviewers must apply to every merge request.

---

## Before Opening a Pull Request

The following must be true before a pull request is opened. The author is responsible for verifying each point.

### Functional Completeness

- [ ] The feature, fix, or change described in the associated ticket is implemented.
- [ ] All acceptance criteria in the ticket are satisfied.
- [ ] No known defects or incomplete edge cases are left unaddressed without a documented follow-up ticket.

### Tests

- [ ] Unit tests (Tier 1) are written for all new or modified production classes.
- [ ] Integration tests (Tier 2) are written for any new use case or cross-component interaction.
- [ ] Contract tests (Tier 3) are written or updated if this PR introduces or modifies a plugin.
- [ ] All tests pass locally.
- [ ] Coverage has not decreased below the project minimum (90% overall, 95% for critical modules).
- [ ] Error paths and edge cases have test coverage; the happy path alone is insufficient.

### Code Quality

- [ ] All public methods and interfaces are documented.
- [ ] No commented-out code is included. Removed code is deleted, not commented out.
- [ ] No debug logging left in production code paths.
- [ ] No hardcoded configuration values; all configurable values are in the configuration object.

### Architecture

- [ ] No layer boundary violations (inner layers do not import outer layers).
- [ ] No concrete infrastructure types instantiated inside Domain or Application Layer classes.
- [ ] If a new extension point is introduced, it is documented in `architecture/03-extensibility.md`.
- [ ] If a new architectural decision is made, an ADR is created and merged in the same PR or ahead of it.

---

## Pull Request Description Template

Every pull request must include a description that covers the following:

```
## Summary
[What does this PR do? One to three sentences.]

## Motivation
[Why is this change needed? Reference the ticket or ADR if applicable.]

## Changes Made
[High-level summary of what was changed, added, or removed. List files only if the scope is small enough that a list is useful.]

## Testing
[What test scenarios were added? What was the coverage before and after?]

## Architecture Notes
[Were any architectural decisions made? Are there any trade-offs or known limitations the reviewer should understand?]

## Follow-up Work
[Any known follow-up tickets or deferred work created as a result of this PR.]
```

---

## Code Review Checklist

This checklist is applied by both the author (self-review before opening) and by reviewers. Every item must be explicitly considered. Unchecked items must be addressed before approval.

### SOLID Compliance

| Check | Applies To |
|-------|-----------|
| Each class has one responsibility and one reason to change (SRP) | All classes |
| New behaviour is added via extension (new class), not by modifying existing tested code (OCP) | Application and Domain Layer |
| All concrete implementations honour the full contract of their interface, including implicit expectations (LSP) | All implementations |
| Interfaces are narrow; no implementation is forced to stub out methods it does not use (ISP) | All interfaces |
| All collaborators are injected via constructor as interface types; no concrete infrastructure instantiated in Domain or Application (DIP) | Domain, Application Layer |

### OOP Concepts

| Check | Applies To |
|-------|-----------|
| Object state is never mutated directly from outside the object; only through methods that enforce invariants (Encapsulation) | All classes |
| Dependencies are expressed as abstractions; domain code does not know about concrete infrastructure (Abstraction) | Domain, Application Layer |
| Inheritance is used only for genuine "is-a" relationships; code reuse is achieved through composition (Inheritance) | All classes |
| Conditional branching on type has been replaced with polymorphism where a second or subsequent type has been added (Polymorphism) | All classes |
| Value objects are immutable; mutations return new instances (Immutability) | Value Objects |

### Architecture Layer Rules

| Check | Applies To |
|-------|-----------|
| No import from an outer layer exists inside an inner layer | Domain, Application Layer |
| No persistence logic in the Domain or Application Layer | Domain, Application Layer |
| No business rules in the Infrastructure or Delivery Layer | Infrastructure, Delivery Layer |
| Delivery Layer methods are thin: parse, map, invoke use case, map result | Delivery Layer |
| Plugins only receive platform services through PluginContext | All plugins |
| Plugin Registry is the only path through which plugins are invoked | Application Layer, core |

### Testing

| Check | Applies To |
|-------|-----------|
| Unit tests follow AAA structure | All unit tests |
| Test names express "given / when / then" | All tests |
| Each test verifies one logical outcome | All tests |
| All collaborators in unit tests are replaced with appropriate test doubles | All unit tests |
| Error paths and edge cases have dedicated tests | All tests |
| No logic (branching, loops) in the assertion section of tests | All tests |
| New plugin has a contract test class extending the base contract test | All plugin tests |
| Test builders are used for complex object construction | Tests with complex setup |
| Coverage has not decreased | All PRs |

### Code Quality and Best Practices

| Check | Applies To |
|-------|-----------|
| Names reveal intent; no abbreviations or vague terms | All code |
| Methods do one thing and are under approximately 20 lines of meaningful code | All methods |
| No method has more than four parameters; complex input uses a parameter object | All methods |
| Exceptions are typed and include context; no generic exceptions thrown | Error handling code |
| No silent exception suppression (empty catch blocks) | All catch blocks |
| Logging is structured; no string concatenation; no sensitive data logged | All log statements |
| No hardcoded configuration values in business logic | Domain, Application Layer |
| All public API is documented with contract, pre-conditions, and exceptions | All public interfaces and classes |

### Documentation

| Check | Applies To |
|-------|-----------|
| If a new extension point was added, `architecture/03-extensibility.md` is updated | Extension point additions |
| If an architectural decision was made, an ADR is created or updated | Architecture changes |
| If a new critical path module was created, coverage configuration is updated | New critical modules |

---

## Reviewer Conduct Standards

- A review must be completed within one business day of the PR being opened and assigned.
- Comments must be specific, constructive, and reference the relevant standard or checklist item.
- Blocking comments must state clearly what must change and why.
- Non-blocking suggestions must be labelled as such ("nit:", "optional:", "consider:").
- Approval means the reviewer has personally verified every applicable checklist item, not merely read the code.

---

## Merge Policy

- A minimum of one approved review from a senior engineer or architect is required.
- All CI checks must pass (unit tests, integration tests, coverage gate, static analysis).
- No unresolved blocking comments may remain at time of merge.
- The author resolves merge conflicts and is responsible for the final merged state.
- Squash merge is the default merge strategy to maintain a clean, linear history.

---

## References

- `adr/ADR-0002` — SOLID Principles Enforcement
- `adr/ADR-0003` — Testing Standards and Coverage Requirements
- `principles/01-solid.md` — SOLID principles reference
- `principles/04-best-practices.md` — Engineering best practices
- `testing/01-testing-standards.md` — Testing standards
- `architecture/04-layered-architecture.md` — Layer rules and boundary violations
