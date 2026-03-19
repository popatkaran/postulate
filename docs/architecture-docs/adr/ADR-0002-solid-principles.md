# ADR-0002: SOLID Principles as Mandatory Engineering Standards

**Status:** Accepted  
**Date:** 2025-01-01  
**Deciders:** Architecture Review Board  
**Ticket / RFC Reference:** RFC-002

---

## Context

As the system grows in complexity — particularly given the plugin-first architecture (ADR-0001) — there is a risk that individual engineering decisions, made in isolation, accumulate into structural debt. Without explicit standards, codebase-wide consistency cannot be enforced in code review, and new team members have no clear framework for design decisions.

SOLID principles, originally formulated by Robert C. Martin, provide a proven, language-agnostic framework for structuring object-oriented and component-based systems. They are especially relevant to a plugin-first, extensible architecture because they govern how components relate to each other and how change is managed without cascading breakage.

This ADR formalises SOLID as a mandatory standard — not a recommendation — and defines what compliance means in the context of this project.

## Decision Drivers

- Code review must have objective, documented criteria for structural correctness.
- Onboarding engineers must be able to learn the expected design model from written standards.
- The plugin contract and core abstractions must be resistant to breaking changes caused by poor separation of concerns.
- Automated static analysis should be able to enforce aspects of these standards where tooling permits.

## Decision

**All production code, test code, and plugin implementations in this project must adhere to the five SOLID principles as defined below. Compliance is a prerequisite for merge approval.**

---

## SOLID Definitions and Project-Specific Interpretation

### S — Single Responsibility Principle (SRP)

**Statement:** A class (or module, or component) should have one, and only one, reason to change.

**Project interpretation:** Each class encapsulates a single, clearly nameable concept. If a class's name requires the word "and" to describe it, it is a candidate for decomposition. A class that handles both data persistence and business rule validation violates SRP.

**Pseudo-code illustration:**

```
// VIOLATION — one class with two reasons to change
class UserService:
    method saveUser(user):
        validate(user)         // reason 1: validation rules change
        database.insert(user)  // reason 2: persistence mechanism changes

// COMPLIANT
class UserValidator:
    method validate(user): ...

class UserRepository:
    method save(user): ...

class UserRegistrationService:
    constructor(validator: UserValidator, repository: UserRepository)
    method register(user):
        validator.validate(user)
        repository.save(user)
```

---

### O — Open/Closed Principle (OCP)

**Statement:** Software entities should be open for extension but closed for modification.

**Project interpretation:** Adding new behaviour must not require modifying existing, tested code. Extension points must be provided via abstractions (interfaces, abstract base types, hook registrations). This principle is foundational to the plugin system: the core must never be modified to accommodate a new plugin.

**Pseudo-code illustration:**

```
// VIOLATION — adding a new export format requires modifying this class
class ReportExporter:
    method export(report, format):
        if format == "pdf":  ...
        if format == "csv":  ...
        // adding "xlsx" requires editing this method

// COMPLIANT — define an abstraction and extend by adding new implementations
interface ExportStrategy:
    method export(report): Bytes

class PdfExportStrategy implements ExportStrategy: ...
class CsvExportStrategy implements ExportStrategy: ...
class XlsxExportStrategy implements ExportStrategy: ...  // no modification to existing code

class ReportExporter:
    constructor(strategy: ExportStrategy)
    method export(report):
        return strategy.export(report)
```

---

### L — Liskov Substitution Principle (LSP)

**Statement:** Objects of a subtype must be substitutable for objects of their supertype without altering the correctness of the program.

**Project interpretation:** If module A depends on abstraction B, any concrete implementation of B must honour the full contract of B — including pre-conditions, post-conditions, and invariants. A subtype must not weaken pre-conditions, strengthen post-conditions, or throw exceptions not declared in the base contract. Plugin implementations must satisfy LSP with respect to the Plugin Contract.

**Pseudo-code illustration:**

```
// VIOLATION — subtype throws an exception that the base contract does not declare
class ReadOnlyRepository extends Repository:
    method save(entity):
        throw UnsupportedOperationException()  // violates contract; callers do not expect this

// COMPLIANT — model the distinction at the abstraction level
interface ReadableRepository:
    method findById(id): Entity

interface WritableRepository extends ReadableRepository:
    method save(entity): void

class InMemoryReadOnlyRepository implements ReadableRepository: ...
class DatabaseRepository implements WritableRepository: ...
```

---

### I — Interface Segregation Principle (ISP)

**Statement:** Clients should not be forced to depend on interfaces they do not use.

**Project interpretation:** Interfaces must be narrow and role-specific. A plugin that only needs to respond to lifecycle events must not be forced to implement data serialisation methods. Fat interfaces are a design smell indicating that the abstraction has multiple responsibilities.

**Pseudo-code illustration:**

```
// VIOLATION — all implementations must provide methods they do not need
interface Worker:
    method doWork(): void
    method generateReport(): Report
    method sendNotification(): void

// COMPLIANT — segregated interfaces; each implementor takes only what it needs
interface Executable:
    method doWork(): void

interface Reportable:
    method generateReport(): Report

interface Notifiable:
    method sendNotification(): void

class BackgroundProcessor implements Executable: ...
class AuditLogger implements Executable, Reportable: ...
```

---

### D — Dependency Inversion Principle (DIP)

**Statement:** High-level modules should not depend on low-level modules. Both should depend on abstractions. Abstractions should not depend on details.

**Project interpretation:** Core domain logic must never import or instantiate concrete infrastructure classes (databases, file systems, HTTP clients, third-party SDKs) directly. All such dependencies are injected via constructor or factory, typed against an interface. This enables test doubles to replace real implementations in all test scenarios.

**Pseudo-code illustration:**

```
// VIOLATION — high-level business logic depends on a concrete database class
class OrderProcessor:
    constructor():
        this.db = new PostgresDatabase()  // direct instantiation, hard to test

    method process(order):
        this.db.save(order)

// COMPLIANT — depend on an abstraction; inject the concrete at the composition root
interface OrderStore:
    method save(order): void

class OrderProcessor:
    constructor(store: OrderStore)
    method process(order):
        store.save(order)

// In production composition root:
processor = new OrderProcessor(new PostgresOrderStore())

// In tests:
processor = new OrderProcessor(new InMemoryOrderStore())
```

---

## Consequences

### Positive

- Consistent design language across the codebase regardless of team member.
- Testability is structurally enforced: DIP ensures test doubles are always possible.
- OCP protects existing tests from breakage when new features are added.

### Negative

- Increased upfront abstraction requires more initial design thought.
- Developers unfamiliar with SOLID require onboarding investment.

### Neutral / Follow-up Actions

- Code review checklist must include a SOLID compliance section (see `guides/contributing.md`).
- Static analysis rules should be configured to flag direct instantiation of infrastructure types in domain modules where tooling supports it.

## Compliance

- This ADR is itself a compliance document. All five principles are mandatory.
- Exceptions require explicit written approval from the Architecture Review Board and must be documented inline with justification.

## References

- `principles/01-solid.md` — Full SOLID reference with extended examples
- `principles/02-oop-concepts.md` — OOP concepts guide
- `guides/contributing.md` — Code review checklist
- Martin, Robert C. — "Agile Software Development, Principles, Patterns, and Practices"
