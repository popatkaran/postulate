# Object-Oriented Programming Concepts Reference

**Document Version:** 1.0  
**Last Updated:** 2025-01-01  
**Owner:** Architecture Review Board  
**Status:** Approved

---

## Purpose

This document defines the core OOP concepts as they are applied in this project and how each concept reinforces the architectural goals of scalability, extensibility, and maintainability. It is not a tutorial; it is a normative reference defining how these concepts must be applied.

Note: All code examples are written in language-agnostic pseudo-code.

---

## Encapsulation

### Definition

Encapsulation is the bundling of data with the operations that act on that data, and the restriction of direct access to an object's internal state from outside that object. The object exposes behaviour, not data.

### Application Rule

An object's fields must never be publicly accessible for direct mutation. All state changes occur through methods that enforce invariants. Getter methods that return internal mutable collections are a form of encapsulation violation — the caller can mutate the collection without the object's knowledge.

```
// VIOLATION — internal state exposed for direct mutation

class ShoppingCart:
    public lines: List<CartLine>  // direct mutation bypasses all business rules

cart.lines.add(new CartLine(...))  // bypasses quantity validation, price rules, etc.

// COMPLIANT — state changes occur only through methods that enforce rules

class ShoppingCart:
    private lines: List<CartLine>
    private maxLineCount: int = 50

    method addLine(line: CartLine): void
        if lines.size() >= maxLineCount:
            raise CartCapacityExceededException()
        if lines.contains(line.productId):
            raise DuplicateProductException()
        lines.add(line)

    method getLines(): List<CartLine>
        return List.immutableCopy(lines)  // defensive copy; caller cannot mutate
```

### Encapsulation and Domain Entities

Domain entities (see `architecture/04-layered-architecture.md`) are the primary users of encapsulation. An entity must never allow its invariants to be broken by external code. The only way to change an entity's state is by calling a method that enforces all relevant business rules.

---

## Abstraction

### Definition

Abstraction is the act of representing essential features without including implementation details. In OOP, this is expressed through interfaces and abstract types that define what an object does without specifying how it does it.

### Application Rule

Every significant dependency in the system must be expressed as an abstraction (interface or abstract type), not as a concrete class. The consumer codes against the abstraction; the concrete implementation is injected at the Composition Root.

This is the direct application of the Dependency Inversion Principle and the mechanism by which the domain layer achieves complete independence from infrastructure.

```
// Abstraction — the domain layer's view of persistence
interface ProductCatalogue:
    method findByCategory(category: Category): List<Product>
    method findById(id: ProductId): Optional<Product>
    method findBySku(sku: SKU): Optional<Product>

// Concrete implementation — lives in Infrastructure, unknown to Domain
class ElasticsearchProductCatalogue implements ProductCatalogue:
    constructor(esClient: ElasticsearchClient)
    method findByCategory(category): List<Product>
        // Elasticsearch-specific query
    ...

// Alternative implementation for tests
class InMemoryProductCatalogue implements ProductCatalogue:
    constructor(initialProducts: List<Product>)
    method findByCategory(category): List<Product>
        return initialProducts.filter(p -> p.category == category)
```

---

## Inheritance

### Definition

Inheritance is a mechanism by which a subtype acquires the properties and behaviour of a supertype. It models an "is-a" relationship.

### Application Rules

Inheritance must be used conservatively and deliberately. The following rules govern its use:

**Rule 1: Prefer composition over inheritance for behaviour reuse.**

Inheritance for the purpose of reusing code creates tight coupling between classes and violates encapsulation (the subtype can access all protected members of the parent). If the goal is code reuse, use composition and delegation instead.

```
// AVOID — inheritance used only for code reuse, not to model a type relationship
class BaseEmailSender:
    method buildMimeMessage(from, to, subject, body): MimeMessage ...

class WelcomeEmailSender extends BaseEmailSender:
    method send(user): void
        msg = buildMimeMessage(noreply, user.email, "Welcome", ...)
        transport.send(msg)

class InvoiceEmailSender extends BaseEmailSender:  // same pattern, brittle coupling

// PREFER — delegate to a shared helper; no inheritance coupling
class MimeMessageBuilder:
    method build(from, to, subject, body): MimeMessage ...

class WelcomeEmailSender:
    constructor(builder: MimeMessageBuilder, transport: EmailTransport)
    method send(user): void
        msg = builder.build(noreply, user.email, "Welcome", ...)
        transport.send(msg)
```

**Rule 2: Use inheritance to model genuine type hierarchies.**

Inheritance is appropriate when a subtype truly is a specialisation of its supertype and must honour its full contract (LSP). Abstract base classes with lifecycle hooks (template method pattern) are an approved use of inheritance.

```
// APPROPRIATE — abstract base provides template; subclasses provide specialisation

abstract class ScheduledJob:
    // Template method — defines the algorithm skeleton
    final method run(): void
        log.info("Job starting", this.getJobName())
        startTime = now()
        try:
            this.execute()    // hook: subclass provides the work
            log.info("Job completed", this.getJobName(), elapsed(startTime))
        catch exception:
            log.error("Job failed", this.getJobName(), exception)
            this.onFailure(exception)  // hook: subclass may override

    abstract method execute(): void
    abstract method getJobName(): string

    // Optional hook with default no-op implementation
    method onFailure(exception: Exception): void
        // default: do nothing

class DailyReportJob extends ScheduledJob:
    method getJobName(): string = "daily-report"
    method execute(): void
        reportService.generateDailyReport()
```

**Rule 3: Limit inheritance hierarchies to a maximum depth of two (excluding framework-mandated base classes).**

Deep inheritance hierarchies are difficult to understand, test, and modify. Beyond two levels, the "is-a" semantic is almost always a rationalisation rather than a genuine model.

---

## Polymorphism

### Definition

Polymorphism is the ability of different types to respond to the same message (method call) in type-specific ways. At runtime, the appropriate implementation is determined by the actual type of the object, not the declared type of the variable.

### Application Rule

Polymorphism is the preferred mechanism for eliminating conditional branching on type. An if-else chain or switch statement that dispatches on a type field is a polymorphism deficiency. Replace it with an interface and multiple implementations.

```
// VIOLATION — conditional dispatch on type; violates OCP and grows with each new type

method renderWidget(widget: Widget): string
    if widget.type == "text":
        return "<p>" + widget.content + "</p>"
    elif widget.type == "image":
        return "<img src='" + widget.url + "' />"
    elif widget.type == "video":  // new requirement: edit required here
        return "<video src='" + widget.url + "'></video>"

// COMPLIANT — polymorphic dispatch; new types added without modification

interface Widget:
    method render(): string

class TextWidget implements Widget:
    constructor(content: string)
    method render(): string = "<p>" + content + "</p>"

class ImageWidget implements Widget:
    constructor(url: URL, altText: string)
    method render(): string = "<img src='" + url + "' alt='" + altText + "' />"

class VideoWidget implements Widget:  // new type: zero modification elsewhere
    constructor(url: URL)
    method render(): string = "<video src='" + url + "'></video>"

// Caller is polymorphic; works with any Widget implementation
method renderAll(widgets: List<Widget>): string
    return widgets.map(w -> w.render()).join("\n")
```

---

## Object Composition

### Definition

Composition is the building of complex objects from simpler ones by establishing "has-a" relationships. A composed object delegates part of its behaviour to its component objects.

### Application Rule

Composition is the primary mechanism for building complex behaviour in this project. It is always preferred over inheritance for code reuse. It results in smaller, more focused classes, each individually testable (SRP). The assembled object delegates to components rather than inheriting their behaviour.

```
// Example: an order processing pipeline composed from focused components

class OrderProcessor:
    constructor(
        validator: OrderValidator,
        pricer: PricingEngine,
        inventoryChecker: InventoryChecker,
        fraudDetector: FraudDetector,
        orderRepository: OrderRepository
    )

    method process(order: DraftOrder): ProcessedOrder
        validatedOrder = validator.validate(order)
        pricedOrder = pricer.apply(validatedOrder)
        inventoryChecker.verify(pricedOrder)
        fraudDetector.screen(pricedOrder)
        processedOrder = pricedOrder.confirm()
        orderRepository.save(processedOrder)
        return processedOrder
```

Each component is independently testable, independently replaceable (DIP), and has a single responsibility (SRP). `OrderProcessor` itself tests only the orchestration, not the logic inside any component.

---

## Immutability

### Definition

An immutable object is one whose state cannot be changed after construction. All mutations produce new objects rather than modifying existing ones.

### Application Rule

Value Objects (in the Domain Layer) must be immutable. Immutable objects are always thread-safe, never require defensive copying, and produce straightforward equality semantics.

```
// Value Object — immutable by design

class Money:
    private readonly amount: Decimal
    private readonly currency: CurrencyCode

    constructor(amount: Decimal, currency: CurrencyCode)
        if amount < 0: raise NegativeMoneyException()
        if currency is null: raise InvalidCurrencyException()
        this.amount = amount
        this.currency = currency

    // Returns a new instance; this instance is unchanged
    method add(other: Money): Money
        if this.currency != other.currency:
            raise CurrencyMismatchException()
        return new Money(this.amount + other.amount, this.currency)

    method multiply(factor: Decimal): Money
        return new Money(this.amount * factor, this.currency)

    // Structural equality for value objects
    method equals(other: Money): boolean
        return this.amount == other.amount and this.currency == other.currency

    method toString(): string = amount + " " + currency
```

---

## References

- `principles/01-solid.md` — SOLID principles (DIP, OCP rely on abstraction; LSP governs inheritance)
- `principles/03-design-patterns.md` — Design patterns that implement these OOP concepts
- `architecture/04-layered-architecture.md` — How encapsulation governs domain entity design
- `testing/01-testing-standards.md` — How composition and DIP enable testability
