# Design Patterns Reference

**Document Version:** 1.0  
**Last Updated:** 2025-01-01  
**Owner:** Architecture Review Board  
**Status:** Approved

---

## Purpose

This document catalogues the design patterns approved for use in this project. For each pattern, it defines: the problem it solves, when to use it, when not to use it, a pseudo-code illustration, and how it relates to the architectural principles of this project.

Patterns are grouped by category: Creational, Structural, and Behavioural.

Note: All code examples are written in language-agnostic pseudo-code.

---

## Creational Patterns

### Factory Method

**Problem:** A class needs to create an object but should not need to know the concrete type of the object it is creating.

**When to use:** When the type of object to create is determined by runtime data, configuration, or a registered strategy. When creating an object directly would violate DIP.

**When not to use:** When there is only one implementation and no expectation of variation. Unnecessary factories add complexity without benefit.

```
// Pseudo-code

interface NotificationChannel:
    method send(recipient: string, message: string): void

class EmailChannel implements NotificationChannel: ...
class SmsChannel implements NotificationChannel: ...
class PushChannel implements NotificationChannel: ...

class NotificationChannelFactory:
    private channels: Map<string, NotificationChannel>

    method register(type: string, channel: NotificationChannel): void
        channels.put(type, channel)

    method create(type: string): NotificationChannel
        return channels.get(type)
            .orElseThrow(UnknownChannelTypeException(type))
```

---

### Builder

**Problem:** An object requires a large number of parameters to construct, some of which are optional. Direct constructor calls with many parameters are hard to read and error-prone.

**When to use:** When constructing complex objects with many optional attributes. Value objects with more than three or four fields. Test data construction (test builder pattern).

**When not to use:** Simple objects with two or fewer required, non-optional parameters.

```
// Pseudo-code

class QueryBuilder:
    private table: string
    private conditions: List<Condition> = []
    private orderBy: List<string> = []
    private limit: Optional<int> = empty
    private offset: int = 0

    method from(table: string): QueryBuilder
        this.table = table
        return this

    method where(condition: Condition): QueryBuilder
        conditions.add(condition)
        return this

    method orderBy(field: string, direction: SortDirection): QueryBuilder
        orderBy.add(field + " " + direction)
        return this

    method limit(n: int): QueryBuilder
        this.limit = Optional.of(n)
        return this

    method offset(n: int): QueryBuilder
        this.offset = n
        return this

    method build(): Query
        if table is null: raise QueryBuilderException("table is required")
        return new Query(table, conditions, orderBy, limit, offset)

// Usage — readable, parameter names are self-documenting
query = new QueryBuilder()
    .from("orders")
    .where(Condition.eq("status", "ACTIVE"))
    .where(Condition.gte("total", 100))
    .orderBy("createdAt", SortDirection.DESC)
    .limit(20)
    .offset(40)
    .build()
```

---

### Singleton (Restricted Use)

**Problem:** Some resources (configuration, connection pools) should have exactly one instance in the application.

**When to use:** Only for stateless, immutable configuration objects or resources that are genuinely singular in nature (e.g., a thread pool). **This is a restricted pattern. See the warning below.**

**Warning:** Singletons that hold mutable state violate ADR-0004 (Stateless Core). Singletons that are globally accessible violate DIP (you cannot inject a test double). The only acceptable form of singleton in this project is an immutable object managed by the Dependency Injection container and injected like any other dependency.

```
// PROHIBITED — static mutable singleton
class AppConfig:
    static instance: AppConfig
    static method getInstance(): AppConfig
        if instance is null: instance = new AppConfig()
        return instance

// ACCEPTABLE — immutable configuration, managed by DI container
class AppConfig:
    readonly databaseUrl: string
    readonly cacheUrl: string
    readonly maxPlugins: int

    constructor(databaseUrl, cacheUrl, maxPlugins): ...

// The DI container ensures one instance is created and injected everywhere.
// Tests can inject a different AppConfig with test values.
```

---

## Structural Patterns

### Adapter

**Problem:** An existing class has an interface that is incompatible with the interface that a client expects.

**When to use:** Integrating third-party libraries into the system behind a domain-defined interface. Implementing a new Plugin Contract version wrapper for plugins that implement an older contract version.

```
// Pseudo-code

// Domain-defined interface
interface EventPublisher:
    method publish(event: DomainEvent): void

// Third-party library has an incompatible interface
class ThirdPartyMessageBroker:
    method sendMessage(topic: string, payload: bytes, headers: Map): void

// Adapter bridges the gap
class MessageBrokerEventPublisherAdapter implements EventPublisher:
    constructor(broker: ThirdPartyMessageBroker, serialiser: EventSerialiser)

    method publish(event: DomainEvent): void
        payload = serialiser.toBytes(event)
        headers = {"event-type": event.getType(), "trace-id": event.getTraceId()}
        broker.sendMessage(event.getType(), payload, headers)
```

---

### Decorator

**Problem:** Behaviour needs to be added to individual objects without affecting other objects of the same class. Subclassing is impractical because there are too many independent behavioural variations.

**When to use:** Adding cross-cutting concerns (logging, caching, retry, validation) to a service without modifying the service class. Each concern becomes a decorator layer.

```
// Pseudo-code

interface OrderRepository:
    method findById(id: OrderId): Optional<Order>
    method save(order: Order): void

// Core implementation
class SqlOrderRepository implements OrderRepository: ...

// Caching decorator
class CachingOrderRepository implements OrderRepository:
    constructor(delegate: OrderRepository, cache: Cache, ttl: Duration)

    method findById(id: OrderId): Optional<Order>
        cached = cache.get(id)
        if cached is present: return cached
        result = delegate.findById(id)
        result.ifPresent(order -> cache.set(id, order, ttl))
        return result

    method save(order: Order): void
        delegate.save(order)
        cache.invalidate(order.getId())

// Logging decorator
class LoggingOrderRepository implements OrderRepository:
    constructor(delegate: OrderRepository, logger: Logger)

    method findById(id: OrderId): Optional<Order>
        logger.debug("Finding order", id)
        result = delegate.findById(id)
        logger.debug("Order find result", id, result.isPresent())
        return result

    method save(order: Order): void
        logger.debug("Saving order", order.getId())
        delegate.save(order)

// Composition root assembles the layers
repo = new LoggingOrderRepository(
    new CachingOrderRepository(
        new SqlOrderRepository(db),
        cache, Duration.ofSeconds(30)
    ),
    logger
)
```

---

### Facade

**Problem:** A subsystem has a complex interface with many components. Clients should not need to understand the internals of the subsystem.

**When to use:** Providing a simple entry point to a complex subsystem. Simplifying the interface exposed to the Delivery Layer.

```
// Pseudo-code

// Complex subsystem
class OrderFacade:
    constructor(
        orderCreationService,
        inventoryService,
        pricingService,
        paymentService,
        notificationService
    )

    method placeOrder(command: PlaceOrderCommand): OrderId
        draft = orderCreationService.createDraft(command)
        inventoryService.reserve(draft)
        pricedOrder = pricingService.applyPricing(draft)
        payment = paymentService.charge(pricedOrder, command.paymentDetails)
        confirmedOrder = pricedOrder.confirm(payment)
        orderCreationService.persist(confirmedOrder)
        notificationService.confirmToCustomer(confirmedOrder)
        return confirmedOrder.getId()
```

---

### Proxy

**Problem:** Access to an object needs to be controlled, logged, or augmented without the client knowing.

**When to use:** Lazy initialisation of expensive resources, access control enforcement, telemetry injection. Preferred over inheritance for these concerns.

---

## Behavioural Patterns

### Strategy

**Problem:** A family of algorithms exists and must be interchangeable. The algorithm should be selectable at runtime.

**When to use:** Payment methods, export formats, sorting algorithms, routing rules, validation strategies. Any situation where a behaviour varies by type or configuration.

**Relationship to OCP:** The Strategy pattern is the primary mechanism for implementing OCP. A new strategy class is added; no existing code changes.

```
// Pseudo-code

interface PricingStrategy:
    method calculatePrice(product: Product, customer: Customer): Money

class StandardPricingStrategy implements PricingStrategy:
    method calculatePrice(product, customer): Money
        return product.basePrice

class VolumePricingStrategy implements PricingStrategy:
    method calculatePrice(product, customer): Money
        if customer.orderHistory.totalVolume > 1000:
            return product.basePrice.multiply(0.9)
        return product.basePrice

class PremiumMemberPricingStrategy implements PricingStrategy:
    method calculatePrice(product, customer): Money
        if customer.membershipTier == PREMIUM:
            return product.basePrice.multiply(0.85)
        return product.basePrice

class PricingEngine:
    constructor(strategies: List<PricingStrategy>)

    method bestPriceFor(product: Product, customer: Customer): Money
        return strategies
            .map(s -> s.calculatePrice(product, customer))
            .min()
```

---

### Observer

**Problem:** When one object changes state, an open-ended set of other objects need to be notified without the subject being tightly coupled to its observers.

**When to use:** Domain event publication. Extension point fanout in the plugin system. UI event handling.

**Relationship to the Plugin System:** The extension point mechanism (see `architecture/03-extensibility.md`) is an implementation of the Observer pattern.

```
// Pseudo-code

interface EventListener<T extends DomainEvent>:
    method onEvent(event: T): void

class EventBus:
    private listeners: Map<EventType, List<EventListener>>

    method subscribe(type: EventType, listener: EventListener): void
        listeners.get(type).add(listener)

    method publish(event: DomainEvent): void
        for each listener in listeners.get(event.getType()):
            try:
                listener.onEvent(event)
            catch exception:
                log.error("Event listener failed", listener, exception)
                // Isolated: other listeners still receive the event
```

---

### Chain of Responsibility

**Problem:** A request needs to pass through a series of handlers, any one of which may process it or pass it to the next.

**When to use:** Request processing pipelines (middleware). Validation chains. Command processing pipelines.

```
// Pseudo-code

interface RequestHandler:
    method setNext(handler: RequestHandler): RequestHandler
    method handle(request: Request): Optional<Response>

abstract class BaseRequestHandler implements RequestHandler:
    private next: Optional<RequestHandler>

    method setNext(handler: RequestHandler): RequestHandler
        this.next = Optional.of(handler)
        return handler

    method passToNext(request: Request): Optional<Response>
        return next.map(h -> h.handle(request))

class AuthenticationHandler extends BaseRequestHandler:
    method handle(request: Request): Optional<Response>
        if not request.hasValidToken():
            return Optional.of(Response.unauthorized())
        return passToNext(request)

class RateLimitHandler extends BaseRequestHandler:
    method handle(request: Request): Optional<Response>
        if rateLimiter.isExceeded(request.clientId):
            return Optional.of(Response.tooManyRequests())
        return passToNext(request)

class BusinessHandler extends BaseRequestHandler:
    method handle(request: Request): Optional<Response>
        result = businessService.process(request)
        return Optional.of(Response.ok(result))

// Assembly
auth = new AuthenticationHandler()
rateLimit = new RateLimitHandler()
business = new BusinessHandler()

auth.setNext(rateLimit).setNext(business)
// Request flows: auth -> rateLimit -> business
```

---

### Template Method

**Problem:** An algorithm has a fixed skeleton, but some steps of the algorithm must vary by subtype.

**When to use:** Plugin lifecycle management. Scheduled job frameworks. Report generation with variable content sections.

**Reference:** See the `ScheduledJob` example in `principles/02-oop-concepts.md`.

---

### Command

**Problem:** A request needs to be encapsulated as an object so it can be parameterised, queued, logged, or undone.

**When to use:** Use cases in the Application Layer are modelled as command/query objects. This pattern decouples the sender of a request from the executor.

```
// Pseudo-code

// Command — an immutable value object representing the intent
class TransferFundsCommand:
    readonly fromAccountId: AccountId
    readonly toAccountId: AccountId
    readonly amount: Money
    readonly idempotencyKey: TransferId

// Handler — executes the command
class TransferFundsHandler:
    constructor(accountRepo: AccountRepository, eventBus: EventBus)

    method handle(command: TransferFundsCommand): void
        fromAccount = accountRepo.findById(command.fromAccountId)
        toAccount = accountRepo.findById(command.toAccountId)
        fromAccount.debit(command.amount)
        toAccount.credit(command.amount)
        accountRepo.save(fromAccount)
        accountRepo.save(toAccount)
        eventBus.publish(new FundsTransferredEvent(command))

// Command Bus — routes commands to handlers
class CommandBus:
    private handlers: Map<CommandType, CommandHandler>

    method dispatch(command: Command): void
        handler = handlers.get(command.getType())
            .orElseThrow(NoHandlerRegisteredException)
        handler.handle(command)
```

---

## Pattern Anti-Patterns

The following are commonly misused patterns that introduce more problems than they solve. They are prohibited unless explicitly approved by the Architecture Review Board.

| Anti-Pattern | Problem | Alternative |
|---|---|---|
| Service Locator | Global registry of services; hides dependencies; untestable | Constructor injection (DIP) |
| God Object | Single class that knows and does too much; violates SRP | Decompose into focused classes |
| Anemic Domain Model | Entities contain only data; all logic in services; defeats encapsulation | Move behaviour into entities |
| Magic Strings / Numbers | Untyped constants scattered in code; no refactoring safety | Typed enumerations and named constants |
| Premature Abstraction | Interface with one implementation, no expected variation | Write the concrete class; extract interface when the second implementation emerges |

---

## References

- `principles/01-solid.md` — SOLID principles that these patterns implement
- `principles/02-oop-concepts.md` — OOP concepts underlying these patterns
- `architecture/03-extensibility.md` — Strategy and Observer patterns in the extension point system
- `architecture/01-plugin-system.md` — Template Method and Observer in the plugin lifecycle
