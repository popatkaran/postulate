# Architecture Documentation Index

**Document Version:** 1.0  
**Last Updated:** 2025-01-01  
**Owner:** Architecture Review Board

---

## How to Use This Index

This index is the master reference for all architecture, principles, testing, and process documentation in this repository. Each entry includes: the document name, a short description of its content, and guidance on when to refer to it.

New engineers should begin with `guides/onboarding.md`, which prescribes a structured reading order.

---

## Architecture Decision Records (ADR)

ADRs capture significant architectural decisions, the context in which they were made, the options considered, and the rationale for the chosen direction. ADRs are immutable records; superseded decisions are marked as deprecated, not deleted.

| Document | Description | When to Refer |
|----------|-------------|---------------|
| `adr/TEMPLATE.md` | Blank ADR template with all required sections and field definitions. | When creating a new ADR for any significant technical decision. |
| `adr/ADR-0001-plugin-first-architecture.md` | Decision to adopt a plugin-first architecture as the primary extensibility model. Covers the Plugin Contract, Plugin Registry, and the rationale for rejecting alternatives. | When questioning why the plugin model exists, or when designing a new capability that could be a plugin. |
| `adr/ADR-0002-solid-principles.md` | Decision to mandate SOLID principles as non-negotiable engineering standards. Includes concise definitions and project-specific interpretation of all five principles. | When a code review dispute arises about design quality, or when onboarding to understand design expectations. |
| `adr/ADR-0003-testing-standards.md` | Decision defining the 90% minimum test coverage requirement and the four-tier testing model (Unit, Integration, Contract, E2E). | When setting up CI coverage gates, or when a question arises about what kind of test to write. |
| `adr/ADR-0004-scalability-strategy.md` | Decision mandating stateless processing units and externalised state for all core and plugin code. | When designing a new component that needs to store state, or when reviewing code for scalability compliance. |

---

## Architecture Documents

Detailed design specifications for each major architectural concern. These documents define what must be built and how. They are the authoritative reference for implementation decisions within their scope.

| Document | Description | When to Refer |
|----------|-------------|---------------|
| `architecture/00-overview.md` | High-level system architecture: the four-layer model, dependency rules, primary control flow, cross-cutting concerns, and the complete document map. | First read for any new engineer. Reference when orienting yourself to where a piece of code belongs. |
| `architecture/01-plugin-system.md` | Complete plugin system design: Plugin Contract definition (with pseudo-code), Plugin Registry behaviour, lifecycle state machine, extension point mechanics, isolation requirements, and versioning policy. | Essential reading before writing, reviewing, or debugging any plugin or plugin infrastructure code. |
| `architecture/02-scalability.md` | Horizontal scaling model, state classification and management rules, all approved scalability patterns (Circuit Breaker, Bulkhead, Rate Limiter, Cache, Async Processing) with pseudo-code, and observability requirements. | When designing components that must scale, handle failure, or emit metrics and logs. |
| `architecture/03-extensibility.md` | Extension point design principles, three approved execution models (Sequential, Parallel, Async), the Extension Points Registry, how to add a new extension point, versioning rules, and anti-patterns. | When adding a new extension point, designing a plugin, or reviewing extensibility decisions. |
| `architecture/04-layered-architecture.md` | Precise rules for each layer (Domain, Application, Infrastructure, Delivery): what each layer contains, what it must not contain, permitted dependencies, boundary violation detection table, and the Composition Root pattern. | When deciding where a new class belongs, or when a code review raises a layer boundary concern. |

---

## Principles Documents

Language-agnostic normative references for the design principles that govern all production code. These documents provide extended examples, edge cases, and application guidance beyond the concise ADR definitions.

| Document | Description | When to Refer |
|----------|-------------|---------------|
| `principles/01-solid.md` | Comprehensive SOLID reference: extended definitions, violation symptoms, correct and incorrect pseudo-code examples for all five principles, and a SOLID compliance checklist for code review. | During design, code review, or when preparing to explain a design decision. The checklist at the end is used in every code review. |
| `principles/02-oop-concepts.md` | How the core OOP concepts — Encapsulation, Abstraction, Inheritance, Polymorphism, Composition, Immutability — are applied in this project. Includes rules, examples, and guidance on when to use inheritance versus composition. | When making structural design decisions: how to model a domain concept, whether to use inheritance or composition, how to expose state safely. |
| `principles/03-design-patterns.md` | Catalogue of approved design patterns (Creational, Structural, Behavioural) with pseudo-code, "when to use / when not to use" guidance, and a table of prohibited anti-patterns with their alternatives. | Before introducing a new pattern, to verify it is approved and understand the recommended implementation. |
| `principles/04-best-practices.md` | Practical engineering standards covering: error handling rules, naming conventions, method design (SRP, parameter limits), code organisation (feature-based packaging), defensive programming, logging standards, and commenting standards. | During code writing and review to verify adherence to practical coding standards beyond structural design. |

---

## Testing Documents

Complete testing standards, coverage requirements, and test implementation patterns.

| Document | Description | When to Refer |
|----------|-------------|---------------|
| `testing/01-testing-standards.md` | Full testing reference: the testing pyramid, four test tiers with scope and speed budgets, AAA structure, naming convention, one-assertion-per-test rule, test independence, test data builders, and coverage measurement policy. | When writing any test. The primary reference for testing questions. |
| `testing/02-coverage-requirements.md` | Coverage minimum thresholds by module, what is and is not counted in coverage measurement, CI pipeline stage configuration, the coverage gate specification, exception process, and coverage trend monitoring. | When configuring CI, investigating a coverage gate failure, or requesting a coverage exclusion. |
| `testing/03-testing-patterns.md` | Test double taxonomy (Dummy, Stub, Fake, Spy, Mock) with definitions, pseudo-code, and a selection guide. Patterns for testing plugin lifecycle, event-driven flows, error paths, and value objects. Test code quality standards. | When choosing the right test double for a collaborator, or when writing tests for complex scenarios involving events, plugins, or error conditions. |

---

## Guides

Process and reference documents for day-to-day engineering work.

| Document | Description | When to Refer |
|----------|-------------|---------------|
| `guides/onboarding.md` | Structured reading order for new engineers, core mental models explained in plain terms, project glossary, and guidance on where to go for help. | First document to read on joining the project. Share with every new team member on day one. |
| `guides/contributing.md` | Complete contribution process: pre-PR checklist, PR description template, full code review checklist covering SOLID, OOP, architecture layer rules, testing, code quality, and documentation. Reviewer conduct standards and merge policy. | Before opening every pull request (author self-review) and during every code review (reviewer checklist). |

---

## Document Maintenance

All documents in this repository are living references. The following rules govern their maintenance:

- ADRs are append-only. A superseded ADR is marked with status "Superseded by ADR-XXXX" and is never deleted.
- Architecture and principles documents are updated when the approved design changes. Changes require Architecture Review Board review.
- The Extension Points Registry in `architecture/03-extensibility.md` must be updated in the same PR that introduces any new extension point.
- This index must be updated in any PR that adds, removes, or renames a document.

---

## Quick Reference: Which Document Answers My Question?

| Question | Document |
|----------|---------|
| Why does this project use a plugin architecture? | `adr/ADR-0001` |
| What SOLID principles are mandatory? | `adr/ADR-0002` and `principles/01-solid.md` |
| What is the minimum test coverage required? | `adr/ADR-0003` and `testing/02-coverage-requirements.md` |
| How do I write a new plugin? | `architecture/01-plugin-system.md` |
| Where does my new class belong in the layer model? | `architecture/04-layered-architecture.md` |
| What design pattern should I use for this problem? | `principles/03-design-patterns.md` |
| What kind of test double should I use? | `testing/03-testing-patterns.md` |
| What must I check before opening a pull request? | `guides/contributing.md` |
| How do I add a new extension point? | `architecture/03-extensibility.md` |
| How is coverage measured and enforced in CI? | `testing/02-coverage-requirements.md` |
| What are the logging standards? | `principles/04-best-practices.md` |
| How should I handle errors and exceptions? | `principles/04-best-practices.md` |
| I am new — where do I start? | `guides/onboarding.md` |
