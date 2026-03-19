# Testing Patterns and Test Doubles

**Document Version:** 1.0  
**Last Updated:** 2025-01-01  
**Owner:** Architecture Review Board  
**Status:** Approved

---

## Purpose

This document defines the approved patterns for writing effective, maintainable tests. It covers test double taxonomy, when to use each type of double, patterns for structuring test code, and strategies for testing the most complex parts of the system: the plugin lifecycle, event-driven flows, and stateless components. All code examples are written in language-agnostic pseudo-code.

---

## Test Double Taxonomy

The term "test double" (coined by Gerard Meszaros) is the umbrella term for any object used in place of a real collaborator in a test. There are five distinct types. Using the precise terminology matters for communication in code review and documentation.

### 1. Dummy

An object that satisfies a parameter type requirement but is never actually used in the test. Passed to a constructor or method only because the language requires a value.

```
// Pseudo-code

test "given valid user, when registered, then user is persisted":
    dummyLogger = new NullLogger()  // required by constructor, not used in this test
    dummyAuditLog = new NullAuditLogger()
    repo = new InMemoryUserRepository()
    service = new UserRegistrationService(repo, dummyLogger, dummyAuditLog)

    service.register(new ValidUser("test@example.com", "SecurePass1!"))

    assert repo.count() == 1
```

### 2. Stub

An object that provides pre-programmed responses to calls made during the test. A stub does not assert; it only supplies data.

```
// Pseudo-code

class StubExchangeRateService implements ExchangeRateService:
    private rates: Map<CurrencyPair, Decimal>

    constructor(rates: Map<CurrencyPair, Decimal>)
        this.rates = rates

    method getRate(from: Currency, to: Currency): Decimal
        pair = CurrencyPair.of(from, to)
        return rates.get(pair)
            .orElseThrow(UnknownCurrencyPairException(pair))

// Usage in test
exchangeRateStub = new StubExchangeRateService({
    CurrencyPair.of(EUR, USD): 1.085,
    CurrencyPair.of(USD, EUR): 0.922
})
service = new CurrencyConverter(exchangeRateStub)

result = service.convert(Money.of(100, EUR), USD)
assert result == Money.of(108.5, USD)
```

### 3. Fake

A working implementation with simplified behaviour, typically operating in memory rather than on a real infrastructure component. Fakes are stateful and can be shared across a test scenario.

Fakes are the preferred double type for repositories, event buses, caches, and state stores. They are more reliable than mocks for stateful interactions and produce more realistic test behaviour.

```
// Pseudo-code — Fake repository

class InMemoryOrderRepository implements OrderRepository:
    private store: Map<OrderId, Order> = {}

    method findById(id: OrderId): Optional<Order>
        return Optional.ofNullable(store.get(id))

    method save(order: Order): void
        store.put(order.getId(), order)

    method delete(id: OrderId): void
        store.remove(id)

    // Test helper methods — not on the OrderRepository interface
    method count(): int = store.size()
    method contains(id: OrderId): boolean = store.containsKey(id)
    method all(): List<Order> = List.copyOf(store.values())
```

### 4. Spy

A spy is a wrapper around a real object (or a fake) that records how it was used, allowing the test to assert on interactions after the fact.

```
// Pseudo-code — Spy on an EventPublisher

class SpyEventPublisher implements EventPublisher:
    private publishedEvents: List<DomainEvent> = []
    private delegate: EventPublisher

    constructor(delegate: EventPublisher)
        this.delegate = delegate

    method publish(event: DomainEvent): void
        publishedEvents.add(event)  // record
        delegate.publish(event)     // delegate to real behaviour

    // Assertion helpers
    method publishedCount(): int = publishedEvents.size()
    method lastPublished(): DomainEvent = publishedEvents.last()
    method wasPublished(type: EventType): boolean
        return publishedEvents.any(e -> e.getType() == type)
```

### 5. Mock

A mock is an object pre-configured with expectations about how it will be called. If the expectations are not met, the test fails. Mocks are strict: unexpected calls may also cause test failure.

**Use mocks sparingly.** Mocks that assert on internal method calls couple tests to implementation, making them brittle when refactoring. Prefer fakes and spies for stateful collaborators. Reserve mocks for the specific case where the interaction itself is the behaviour being verified and there is no observable state to assert on.

```
// Pseudo-code — appropriate use of a mock

test "given a critical alert, when processed, then the pager service is called once":

    // The interaction IS the outcome; there is no observable state to check
    pagerMock = Mock(PagerService)
    pagerMock.expect("page", called: exactly(1), with: contains("CRITICAL"))

    alertProcessor = new AlertProcessor(pagerMock)
    alertProcessor.process(new Alert("CRITICAL", "Database unreachable"))

    pagerMock.verify()  // assert all expectations were met
```

---

## Test Double Selection Guide

| Collaborator Type | Preferred Double | Rationale |
|---|---|---|
| Simple dependencies not called in test | Dummy | No behaviour needed |
| Read-only data sources (config, reference data) | Stub | Returns fixed data; simple to set up |
| Stateful stores (repositories, caches, queues) | Fake | Stateful; more realistic than a mock |
| Observers, event listeners (notification, audit) | Spy | Verify something was called without breaking encapsulation |
| External I/O where the call itself is the outcome | Mock | When interaction verification is the only option |

---

## Patterns for Testing Specific Concerns

### Testing Plugin Lifecycle

Plugin lifecycle tests fall into the Tier 3 contract test category. The following pattern illustrates how to test lifecycle transitions deterministically.

```
// Pseudo-code — plugin lifecycle transition test

test "given a plugin that fails onInitialise, Registry marks it FAILED and continues":

    // Arrange
    failingPlugin = new BrokenPlugin()  // onInitialise() throws BrokenPluginException
    goodPlugin = new HealthyPlugin()
    registry = new PluginRegistry(ContractVersion.CURRENT)
    registry.register(failingPlugin)
    registry.register(goodPlugin)

    // Act
    registry.startAll()

    // Assert — failing plugin is FAILED, good plugin is still ACTIVE
    assert registry.getStatus(failingPlugin.getId()) == PluginStatus.FAILED
    assert registry.getStatus(goodPlugin.getId()) == PluginStatus.ACTIVE

// Pseudo-code — testing plugin isolation during dispatch

test "given a plugin that throws during dispatch, other plugins still receive the event":

    // Arrange
    throwingPlugin = new ThrowingOnDispatchPlugin()
    recordingPlugin = new RecordingPlugin()
    registry = new PluginRegistry(ContractVersion.CURRENT)
    registry.register(throwingPlugin)
    registry.register(recordingPlugin)
    registry.startAll()

    // Act
    registry.dispatch(ExtensionPoints.ON_ENTITY_CREATED, new TestEntity())

    // Assert — recording plugin received the dispatch despite the throwing plugin
    assert recordingPlugin.receivedDispatchCount() == 1
```

### Testing Event-Driven Flows

```
// Pseudo-code — testing that a use case publishes the correct domain event

test "given a new order, when confirmed, then OrderConfirmedEvent is published":

    // Arrange
    eventBusSpy = new SpyEventPublisher(new InMemoryEventBus())
    orderRepo = new InMemoryOrderRepository()
    handler = new ConfirmOrderHandler(orderRepo, eventBusSpy)

    order = OrderTestBuilder.aDraftOrder().withId("ord-99").build()
    orderRepo.save(order)

    // Act
    handler.handle(new ConfirmOrderCommand(OrderId.of("ord-99")))

    // Assert
    assert eventBusSpy.wasPublished(EventType.ORDER_CONFIRMED)
    event = eventBusSpy.lastPublished()
    assert event.getOrderId() == OrderId.of("ord-99")
```

### Testing Error Paths

Every public method with error conditions must have at least one test for each error path. Do not rely on happy-path tests to achieve branch coverage on error handling code.

```
// Pseudo-code — error path tests

test "given an order that does not exist, when add line is attempted, then OrderNotFoundException is raised":

    repo = new InMemoryOrderRepository()  // empty; no order with this id
    handler = new AddOrderLineHandler(repo, new NoOpPluginRegistry())

    command = new AddOrderLineCommand(
        orderId = OrderId.of("non-existent"),
        productId = ProductId.of("prod-1"),
        quantity = Quantity.of(1),
        unitPrice = Money.of(10, "EUR")
    )

    assertRaises(OrderNotFoundException):
        handler.handle(command)

test "given a confirmed order, when add line is attempted, then OrderNotModifiableException is raised":

    repo = new InMemoryOrderRepository()
    confirmedOrder = OrderTestBuilder.aConfirmedOrder().withId("ord-5").build()
    repo.save(confirmedOrder)

    handler = new AddOrderLineHandler(repo, new NoOpPluginRegistry())
    command = new AddOrderLineCommand(OrderId.of("ord-5"), ...)

    assertRaises(OrderNotModifiableException):
        handler.handle(command)
```

### Testing Value Objects

Value objects are some of the easiest classes to test thoroughly because they have no external dependencies.

```
// Pseudo-code — value object tests

test "two Money instances with same amount and currency are equal":
    a = Money.of(100, "EUR")
    b = Money.of(100, "EUR")
    assert a == b

test "two Money instances with different currencies are not equal":
    a = Money.of(100, "EUR")
    b = Money.of(100, "USD")
    assert a != b

test "adding two Money instances in same currency produces correct sum":
    a = Money.of(40, "EUR")
    b = Money.of(35, "EUR")
    result = a.add(b)
    assert result == Money.of(75, "EUR")

test "adding two Money instances in different currencies raises CurrencyMismatchException":
    a = Money.of(40, "EUR")
    b = Money.of(35, "USD")
    assertRaises(CurrencyMismatchException):
        a.add(b)

test "Money with negative amount raises NegativeMoneyException":
    assertRaises(NegativeMoneyException):
        Money.of(-1, "EUR")
```

---

## Test Code Quality Standards

Test code is production code. It must be held to the same structural quality standards.

### Apply SRP to Test Classes

A test class should test one production class. If a test class is testing two unrelated classes, split it.

### Apply DRY Carefully

Removing duplication in test code is desirable, but not at the cost of test readability. A test that is hard to read because its setup is buried in shared helpers is harder to maintain than one with some setup duplication. Shared helpers must be clearly named and must not obscure the essential arrangement of each test.

### Keep Tests Focused

A test method must be readable in under 30 seconds by a developer unfamiliar with the system. If it takes longer, the test is too complex. Simplify the scenario, extract helpers, or rename for clarity.

### No Logic in Tests

Tests must not contain conditional branching (if/else), loops, or exception handling within the assertion section. If a test contains logic, it becomes necessary to test the test, which is impractical.

```
// VIOLATION — logic in test assertion
test "all products in catalogue have valid prices":
    products = catalogue.findAll()
    for each product in products:
        if product.category == "premium":
            assert product.price > Money.of(100, "EUR")  // conditional in test
        else:
            assert product.price > Money.of(0, "EUR")

// COMPLIANT — separate tests for each category
test "premium products have price above 100 EUR":
    premiumProducts = catalogue.findByCategory("premium")
    premiumProducts.forEach(p -> assert p.price > Money.of(100, "EUR"))

test "standard products have positive prices":
    standardProducts = catalogue.findByCategory("standard")
    standardProducts.forEach(p -> assert p.price.isPositive())
```

---

## References

- `testing/01-testing-standards.md` — Test tier definitions, naming, coverage requirements
- `testing/02-coverage-requirements.md` — CI gate configuration
- `principles/01-solid.md` — DIP enables injectable test doubles
- `architecture/01-plugin-system.md` — Plugin Contract, basis for contract tests
