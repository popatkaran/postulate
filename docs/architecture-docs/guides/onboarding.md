# Developer Onboarding Guide

**Document Version:** 1.0  
**Last Updated:** 2025-01-01  
**Owner:** Architecture Review Board  
**Status:** Approved

---

## Purpose

This document is the starting point for every new engineer joining the project. It maps out the documents you must read, in which order, and why. It also explains the mental models that underpin the architecture so that individual technical decisions make sense in context.

---

## Reading Order

Read these documents in sequence before writing any production code. Do not skip ahead; each document builds on the previous one.

### Phase 1: Foundation (Day 1)

These documents provide the architectural rationale and the non-negotiable standards. Understanding *why* the architecture is designed as it is will inform every decision you make.

| Document | Why You Need It |
|----------|----------------|
| `architecture/00-overview.md` | The complete system model: layers, dependencies, philosophy. Read this first. |
| `adr/ADR-0001-plugin-first-architecture.md` | The central architectural decision. Explains why the plugin model was chosen and what it means for your daily work. |
| `adr/ADR-0002-solid-principles.md` | The mandatory engineering standards. Every code review will reference these. |
| `adr/ADR-0003-testing-standards.md` | The testing minimum bar and what is expected of you before merging any code. |
| `adr/ADR-0004-scalability-strategy.md` | Why the system must be stateless and what that means in practice. |

### Phase 2: Design Principles (Day 2)

These documents expand on the rationale documents with detailed guidance and examples.

| Document | Why You Need It |
|----------|----------------|
| `principles/01-solid.md` | Extended SOLID reference with examples. You will refer back to this during design and code review. |
| `principles/02-oop-concepts.md` | How OOP concepts (encapsulation, abstraction, composition) are applied in this project. |
| `principles/03-design-patterns.md` | The approved design patterns. Before reaching for a pattern, check it is listed here. |
| `principles/04-best-practices.md` | Practical coding standards: error handling, naming, method design, logging. |

### Phase 3: Architecture Details (Day 2-3)

These documents define the detailed design of specific architectural concerns.

| Document | Why You Need It |
|----------|----------------|
| `architecture/01-plugin-system.md` | The complete plugin runtime design. Essential if you are writing or reviewing plugin code. |
| `architecture/02-scalability.md` | Scalability patterns, stateless design, observability. |
| `architecture/03-extensibility.md` | How extension points work and how to add new ones. |
| `architecture/04-layered-architecture.md` | Precise rules for each layer. Essential for understanding what code belongs where. |

### Phase 4: Testing and Process (Day 3)

| Document | Why You Need It |
|----------|----------------|
| `testing/01-testing-standards.md` | The complete testing reference: test tiers, AAA structure, naming, contract tests. |
| `testing/02-coverage-requirements.md` | How coverage is measured and enforced in CI. |
| `testing/03-testing-patterns.md` | Test double taxonomy and patterns for testing complex scenarios. |
| `guides/contributing.md` | The pull request process and the full code review checklist. Read before opening your first PR. |

---

## Core Mental Models

### The Dependency Rule

Every architectural decision in this project is downstream of one rule: **dependencies point inward**. The Domain Layer knows nothing about Infrastructure, Delivery, or plugins. This single rule is what makes the system testable, replaceable, and resistant to cascading change.

When you are writing code and you find yourself wanting to import a database driver into a domain class, the Dependency Rule is telling you to stop and introduce an interface.

### Extend, Do Not Modify

The plugin system exists so that new capabilities can be added without touching existing, tested code. If you find yourself adding a new `if` branch to a core class to handle a new type of thing, that is a signal to create a plugin or add a new implementation of an existing interface instead.

### Everything is a Collaborator

No class in this system should be responsible for creating its own dependencies. If a class needs a repository, it receives one through its constructor. This makes every class independently testable and makes the system's dependency graph explicit and visible.

### Statelessness is Not Optional

The platform scales by running multiple instances. Any state held in-process on one instance is invisible to another. If you need to store something that survives a request boundary, it goes into an external store accessed through an injected interface.

---

## Glossary

| Term | Definition |
|------|-----------|
| ADR | Architecture Decision Record. A short document capturing a significant architectural decision, its context, the options considered, and the rationale for the choice made. |
| Composition Root | The single location in the application where all concrete implementations are instantiated and wired together. Typically the application entry point. |
| Domain Event | An immutable record of something that has occurred within the business domain. Named in the past tense. |
| Extension Point | A named, typed hook in the core at which plugins can inject behaviour. Declared by the core; implemented by plugins. |
| Fake | A test double that provides a working, in-memory implementation of an interface, used in place of the real infrastructure implementation in tests. |
| Plugin Contract | The formal interface that all plugins must implement. The sole coupling point between the core and any plugin. |
| Plugin Registry | The central runtime catalogue of all registered plugins. The only mechanism through which the core invokes plugin behaviour. |
| SOLID | Five object-oriented design principles: Single Responsibility, Open/Closed, Liskov Substitution, Interface Segregation, Dependency Inversion. Mandatory standards in this project (ADR-0002). |
| Test Double | Any object used in place of a real collaborator in a test. Includes: Dummy, Stub, Fake, Spy, Mock. |
| Value Object | An immutable domain object with no identity, defined entirely by its attributes. Two value objects with the same attributes are equal. |

---

## Getting Help

- **Architecture questions:** Raise in the Architecture Review Board channel or open a discussion thread linked to the relevant document.
- **Testing questions:** Refer first to `testing/01-testing-standards.md` and `testing/03-testing-patterns.md`. If the answer is not there, ask a senior engineer.
- **Plugin development questions:** Refer to `architecture/01-plugin-system.md`. The Plugin Contract definition and the contract test base class are the definitive references.
- **Code review disagreements:** Reference the relevant checklist item in `guides/contributing.md` or ADR. Design decisions must be grounded in documented principles, not personal preference.

---

## References

This document is the entry point. All other documents are referenced within the reading order above.

For the complete index of all documentation, see `INDEX.md`.
