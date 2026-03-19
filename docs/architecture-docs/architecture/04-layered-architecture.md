# Layered Architecture and Dependency Boundaries

**Document Version:** 1.0  
**Last Updated:** 2025-01-01  
**Owner:** Architecture Review Board  
**Status:** Approved

---

## Purpose

This document defines the precise rules governing the layered architecture of the platform: what each layer is responsible for, what it is not responsible for, how dependencies between layers must be structured, and what constitutes a boundary violation. Compliance with these rules is verified during code review.

---

## The Dependency Rule

The single most important rule of the layered architecture is:

**Source code dependencies must point inward. An inner layer must never name, import, or instantiate anything from an outer layer.**

```
Inward direction of allowed dependencies:

Delivery Layer  -->  Application Layer  -->  Domain Layer
                                          <--  Infrastructure Layer (via interfaces)

Plugin Layer interacts with:
    Application Layer (through extension points)
    Domain Layer (through injected interfaces via PluginContext only)
```

A violation of the dependency rule creates tight coupling that makes the system resistant to change, difficult to test, and structurally fragile.

---

## Layer Definitions and Permitted Dependencies

### Domain Layer

**Contains:** Entities, Value Objects, Domain Services, Domain Events, Repository Interfaces, Domain Exceptions.

**Permitted dependencies:** None. The domain layer has zero external dependencies. It does not import from any framework, any persistence library, or any other layer.

**Must not contain:**
- Imports from the Infrastructure Layer (database drivers, ORM annotations that affect logic, HTTP clients).
- Imports from the Application Layer.
- Imports from the Delivery Layer.
- Framework-specific annotations that carry runtime behaviour (e.g., transaction management annotations that affect business logic).
- Input validation that depends on an external library (validation logic is pure code in the Domain Layer).

**Test approach:** Domain Layer is the easiest to test. No mocks required for unit tests; only pure instances of domain objects.

```
// Pseudo-code — valid Domain Layer class

class Order:  // Entity
    private id: OrderId
    private lines: List<OrderLine>
    private status: OrderStatus
    private events: List<DomainEvent>

    // Business rule enforced by the entity itself
    method addLine(line: OrderLine): void
        if this.status != OrderStatus.DRAFT:
            raise OrderNotDraftException(this.id)
        lines.add(line)
        events.add(new OrderLineAddedEvent(this.id, line))

    method collectEvents(): List<DomainEvent>
        collected = List.copy(events)
        events.clear()
        return collected

// Repository interface lives in the Domain Layer
interface OrderRepository:
    method findById(id: OrderId): Optional<Order>
    method save(order: Order): void
    method delete(id: OrderId): void
```

---

### Application Layer

**Contains:** Use case classes (or command/query handlers), Application Services, DTO (Data Transfer Object) types used at the boundary between Delivery and Application layers, Application-level exceptions.

**Permitted dependencies:** Domain Layer only. No Infrastructure imports. No Delivery imports.

**Must not contain:**
- Business rules (those belong in the Domain Layer).
- Persistence logic (that belongs in Infrastructure).
- HTTP request or response types.
- Framework routing annotations.

**Test approach:** Unit test with in-memory fakes for all repository and service dependencies. No database, no network.

```
// Pseudo-code — Application Layer use case (command handler)

class AddOrderLineHandler:
    constructor(
        orderRepository: OrderRepository,       // Domain interface
        pluginRegistry: PluginRegistry          // Core interface
    )

    method handle(command: AddOrderLineCommand): void
        // 1. Load entity through domain interface
        order = orderRepository.findById(command.orderId)
            .orElseThrow(OrderNotFoundException(command.orderId))

        // 2. Apply business rule on the entity
        line = new OrderLine(command.productId, command.quantity, command.unitPrice)
        order.addLine(line)

        // 3. Dispatch extension point
        pluginRegistry.dispatch(ExtensionPoints.ON_ENTITY_UPDATED, order)

        // 4. Persist through domain interface
        orderRepository.save(order)

        // 5. Publish domain events
        order.collectEvents().forEach(event -> eventBus.publish(event))
```

---

### Infrastructure Layer

**Contains:** Repository implementations, database access code, external HTTP clients, cache adapters, message queue producers and consumers, file system access, email/SMS clients, third-party SDK integrations.

**Permitted dependencies:** Domain Layer (to implement repository interfaces and use entity types), third-party libraries relevant to the infrastructure concern.

**Must not contain:**
- Business rules.
- Application orchestration logic.
- References to Delivery Layer types.

**Key rule:** Infrastructure classes implement interfaces defined in the Domain Layer. The domain defines the contract; infrastructure fulfils it.

```
// Pseudo-code — Infrastructure Layer repository implementation

class SqlOrderRepository implements OrderRepository:  // implements Domain interface
    constructor(connection: DatabaseConnection, mapper: OrderMapper)

    method findById(id: OrderId): Optional<Order>
        row = connection.query(
            "SELECT * FROM orders WHERE id = ?", [id.value]
        )
        if row is empty: return Optional.empty()
        return Optional.of(mapper.toDomain(row))

    method save(order: Order): void
        connection.execute(
            "INSERT INTO orders ... ON CONFLICT UPDATE ...",
            mapper.toRecord(order)
        )
```

---

### Delivery Layer

**Contains:** HTTP controllers/handlers, gRPC service implementations, CLI command parsers, message queue consumer entry points, request-to-command mappers, result-to-response mappers.

**Permitted dependencies:** Application Layer (to invoke use cases). Domain Layer only for entity ID types used in URL path parameters or similar.

**Must not contain:**
- Business rules.
- Direct repository access.
- Any logic beyond: parse input, validate format, invoke use case, map result, return response.

```
// Pseudo-code — Delivery Layer HTTP handler

class OrderController:
    constructor(addOrderLineHandler: AddOrderLineHandler)

    method handleAddLine(httpRequest: HttpRequest): HttpResponse
        // 1. Parse and validate format of input
        body = parseJson(httpRequest.body, AddOrderLineRequestBody)
        if body is invalid: return HttpResponse.badRequest(body.validationErrors)

        // 2. Map to a protocol-agnostic command
        command = new AddOrderLineCommand(
            orderId = OrderId.of(httpRequest.pathParam("orderId")),
            productId = ProductId.of(body.productId),
            quantity = Quantity.of(body.quantity),
            unitPrice = Money.of(body.unitPrice, body.currency)
        )

        // 3. Invoke use case
        try:
            addOrderLineHandler.handle(command)
            return HttpResponse.ok()
        catch OrderNotFoundException:
            return HttpResponse.notFound()
        catch OrderNotDraftException as e:
            return HttpResponse.conflict(e.getMessage())
```

---

## Dependency Injection and the Composition Root

The concrete implementations of all interfaces are wired together at the Composition Root. The Composition Root is the only place in the application where concrete types from the Infrastructure Layer are instantiated and injected into Application Layer classes.

The Composition Root is typically the application entry point (main function or application startup module).

```
// Pseudo-code — Composition Root (application startup)

method bootstrap():
    // Infrastructure
    db = new DatabaseConnection(config.databaseUrl)
    orderRepo = new SqlOrderRepository(db, new OrderMapper())
    cache = new DistributedCacheAdapter(config.cacheUrl)
    cachedOrderRepo = new CachedRepository(orderRepo, cache, Duration.ofSeconds(60))

    // Application
    eventBus = new LocalEventBus()
    pluginRegistry = new PluginRegistry(ContractVersion.CURRENT)

    handler = new AddOrderLineHandler(cachedOrderRepo, pluginRegistry)

    // Plugin registration
    pluginRegistry.register(new AuditPlugin(db))
    pluginRegistry.register(new NotificationPlugin(emailClient))
    pluginRegistry.startAll()

    // Delivery
    controller = new OrderController(handler)
    server = new HttpServer(config.port)
    server.register("/orders/{orderId}/lines", controller)
    server.start()
```

---

## Boundary Violation Detection

The following are the most common boundary violations. Code review must check for all of them.

| Violation | Symptom | Corrective Action |
|-----------|---------|-------------------|
| Domain imports Infrastructure | Domain class references a database driver type | Extract to an interface in the Domain Layer; implement in Infrastructure |
| Application imports Delivery | Application service receives an HttpRequest parameter | Define an application-level command/query DTO; map in the Delivery Layer |
| Domain instantiates Infrastructure | `new PostgresRepository()` inside a domain service | Inject via constructor using the domain interface type |
| Infrastructure contains business rules | If/else business logic inside a repository implementation | Move logic to an entity or domain service |
| Delivery bypasses Application Layer | Controller calls repository directly | Route through a use case / command handler |
| Plugin bypasses PluginContext | Plugin instantiates a service directly | Plugin must receive all services through `PluginContext` |

---

## References

- `architecture/00-overview.md` — Layer diagram and philosophy
- `principles/01-solid.md` — DIP is the theoretical foundation for the dependency rule
- `ADR-0002` — SOLID Principles Enforcement
- `ADR-0001` — Plugin-First Architecture (Plugin Layer dependency rules)
- `testing/01-testing-standards.md` — How layer boundaries simplify testing
