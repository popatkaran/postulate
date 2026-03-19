# Testing Standards

**Document Version:** 1.0  
**Last Updated:** 2025-01-01  
**Owner:** Architecture Review Board  
**Status:** Approved  
**ADR Reference:** ADR-0003

---

## Purpose

This document defines the complete testing standards for the project. It expands on ADR-0003 with detailed guidance on test structure, test double selection, the testing pyramid, and the contract testing model for the plugin system. All engineers writing production or test code must read this document.

All code examples are written in language-agnostic pseudo-code.

---

## The Testing Pyramid

The testing strategy follows a pyramid model: many fast, focused unit tests at the base; fewer, slower integration tests in the middle; a small number of end-to-end tests at the top. The pyramid shape is intentional: it produces fast CI feedback and good localisation of failures.

```
               /\
              /  \
             / E2E\       <-- few, slow, high confidence
            /------\
           /        \
          / Integration\  <-- moderate number, moderate speed
         /--------------\
        /                \
       /   Unit Tests     \ <-- many, very fast, high isolation
      /____________________\
```

Inverting the pyramid (too many E2E tests, too few unit tests) results in slow CI pipelines, flaky tests, and poor diagnostic precision when a test fails.

---

## Tier 1: Unit Tests

### Definition

A unit test exercises a single class or pure function in isolation. All collaborators are replaced with test doubles.

### Structure: Arrange-Act-Assert

Every unit test follows the Arrange-Act-Assert (AAA) pattern. Each section is separated by a blank line. Comments marking the sections are optional but recommended for readability in complex tests.

```
// Pseudo-code

test "given order in DRAFT status, when a line is added, then the order contains the line":

    // Arrange
    product = new Product(ProductId.of("prod-1"), Money.of(50, "EUR"))
    order = Order.createDraft(OrderId.of("ord-1"), CustomerId.of("cust-1"))

    // Act
    order.addLine(new OrderLine(product.getId(), Quantity.of(2), product.price))

    // Assert
    assert order.getLines().size() == 1
    assert order.getLines().first().productId == product.getId()
```

### Naming Convention

Test names must express the complete scenario. The format "given [precondition], when [action], then [expected outcome]" is the standard.

```
// Examples of well-named tests:
test "given valid credentials, when authenticate is called, then an auth token is returned"
test "given an expired token, when validate is called, then TokenExpiredException is raised"
test "given a plugin in ACTIVE state, when stop is called, then plugin transitions to STOPPED"
test "given a null customer id, when order is created, then NullArgumentException is raised"
```

### One Logical Assertion Per Test

A test must verify one logical outcome. Multiple physical assertions are acceptable when they collectively verify a single logical state.

```
// ACCEPTABLE — multiple assertions, one logical outcome (order line was added correctly)
test "given valid product, when addLine called, then order line is correctly initialised":
    order = Order.createDraft(...)
    order.addLine(new OrderLine(productId, quantity, unitPrice))
    line = order.getLines().first()
    assert line.productId == productId      // all assertions verify the same thing:
    assert line.quantity == quantity        // that the line was added with the correct data
    assert line.unitPrice == unitPrice

// VIOLATION — two independent logical outcomes in one test (split into two tests)
test "registration and notification":
    userService.register(newUser)
    assert userRepo.contains(newUser.email)       // outcome 1: user stored
    assert emailSpy.lastSentTo() == newUser.email // outcome 2: notification sent
    // If notification fails, it obscures whether storage passed or failed
```

### Test Independence

Tests must not rely on shared mutable state or execution order. Each test creates all objects it needs. No test may modify class-level or module-level state that another test reads.

---

## Tier 2: Integration Tests

### Definition

An integration test exercises the collaboration between two or more real components, with only external I/O (databases, network, file systems) replaced by in-process fakes or test containers.

### What to Test at the Integration Level

- Repository implementations against a real (or containerised) database to verify SQL queries and mapping correctness.
- Application Layer use cases with real domain objects and in-memory repository fakes.
- Plugin lifecycle: register, start, dispatch, stop — using a real Plugin Registry and a real (test) plugin implementation.
- Event bus publish-subscribe pipelines.

### Integration Test Example

```
// Pseudo-code — integration test for a use case with in-memory repository

test "given a valid add-line command, when handled, then order is updated and persisted":

    // Arrange — real handler with in-memory repository
    orderRepo = new InMemoryOrderRepository()
    pluginRegistry = new NoOpPluginRegistry()  // real registry, no plugins registered
    handler = new AddOrderLineHandler(orderRepo, pluginRegistry)

    order = Order.createDraft(OrderId.of("ord-42"), CustomerId.of("cust-7"))
    orderRepo.save(order)

    command = new AddOrderLineCommand(
        orderId = order.getId(),
        productId = ProductId.of("prod-9"),
        quantity = Quantity.of(3),
        unitPrice = Money.of(25, "EUR")
    )

    // Act
    handler.handle(command)

    // Assert
    stored = orderRepo.findById(order.getId()).get()
    assert stored.getLines().size() == 1
    assert stored.getLines().first().quantity == Quantity.of(3)
```

---

## Tier 3: Contract Tests

### Purpose

Contract tests verify that a concrete plugin implementation correctly satisfies the Plugin Contract (`architecture/01-plugin-system.md`). They are the automated enforcement of the Liskov Substitution Principle for the plugin system.

### How Contract Tests Work

A base contract test class is written once. It defines all the behaviours required by the Plugin Contract. Each plugin provides its own test subclass that supplies a concrete instance of the plugin under test. The full contract test suite runs against every concrete implementation.

```
// Pseudo-code — base contract test class

abstract class PluginContractTest:

    // Subclasses provide the concrete plugin under test
    abstract method createPlugin(): Plugin

    test "plugin has a non-null, non-empty identifier":
        plugin = createPlugin()
        assert plugin.getId() is not null
        assert plugin.getId().value is not empty

    test "plugin declares a contract version":
        plugin = createPlugin()
        assert plugin.getContractVersion() is not null

    test "plugin onInitialise completes without exception given valid context":
        plugin = createPlugin()
        context = new TestPluginContext()
        plugin.onInitialise(context)
        // no exception raised = pass

    test "plugin transitions to healthy state after successful initialisation":
        plugin = createPlugin()
        context = new TestPluginContext()
        plugin.onInitialise(context)
        plugin.onStart()
        assert plugin.getHealth() == HealthStatus.HEALTHY

    test "plugin onStop completes without exception after start":
        plugin = createPlugin()
        context = new TestPluginContext()
        plugin.onInitialise(context)
        plugin.onStart()
        plugin.onStop()
        // no exception raised = pass

    test "plugin getDependencies returns a non-null list":
        plugin = createPlugin()
        assert plugin.getDependencies() is not null

// Concrete plugin's test class
class AuditPluginContractTest extends PluginContractTest:
    method createPlugin(): Plugin
        return new AuditPlugin(new InMemoryAuditLogStore())

    // Additional tests specific to AuditPlugin's own behaviour go here
    test "given AFTER_REQUEST_PROCESSED event, audit log contains the event":
        ...
```

### Coverage Requirement for Contract Tests

Every plugin implementation must pass the full base contract test suite. Plugin-specific test classes must also exercise all extension points the plugin handles, covering both happy path and error path scenarios.

---

## Tier 4: End-to-End Tests

### Definition

An end-to-end test verifies system behaviour from the external entry point (HTTP endpoint, message queue) to the final observable outcome (database record, outgoing email, response body).

### Scope and Constraints

- E2E tests run against a fully assembled system, typically using a containerised infrastructure stack.
- E2E tests are slow and are reserved for the most critical user journeys.
- E2E tests must not overlap in coverage with integration tests. If a scenario can be covered at the integration level, it must be.
- E2E tests must be deterministic: the test must control all inputs and clean up all state after each run.

### When to Write E2E Tests

Write an E2E test only for:
- Critical path journeys: the sequences of actions whose failure would constitute a P1 production incident.
- Cross-plugin interaction verification that cannot be expressed at the integration level.
- Release smoke tests: a minimal set of checks run on every deployment to verify the system starts and serves traffic correctly.

---

## Test Data Management

### Rule: Use Builders for Test Data

Test data construction must use builder patterns or factory methods. Inline construction of complex objects in tests creates brittle, verbose setup code that obscures the test intent.

```
// Pseudo-code — test data builder

class OrderTestBuilder:
    private id: OrderId = OrderId.of("default-test-order")
    private customerId: CustomerId = CustomerId.of("default-test-customer")
    private lines: List<OrderLine> = []
    private status: OrderStatus = OrderStatus.DRAFT

    static method aDraftOrder(): OrderTestBuilder
        return new OrderTestBuilder()

    method withId(id: string): OrderTestBuilder
        this.id = OrderId.of(id)
        return this

    method withLine(productId: string, qty: int, price: decimal): OrderTestBuilder
        this.lines.add(new OrderLine(ProductId.of(productId), Quantity.of(qty), Money.of(price, "EUR")))
        return this

    method build(): Order
        order = Order.createDraft(this.id, this.customerId)
        lines.forEach(l -> order.addLine(l))
        return order

// Usage in tests — intent is clear
order = OrderTestBuilder.aDraftOrder()
    .withId("ord-test-1")
    .withLine("prod-A", 2, 49.99)
    .withLine("prod-B", 1, 9.99)
    .build()
```

### Rule: Tests Must Not Share Mutable Fixtures

If a test modifies an object, that object must be created fresh for each test. Shared fixtures that are mutated by tests cause non-deterministic, order-dependent failures.

---

## Test Coverage Measurement

Coverage must be measured on every CI run. The following dimensions are measured and enforced:

| Dimension | Minimum |
|-----------|---------|
| Line coverage | 90% |
| Branch coverage | 90% |
| Function / method coverage | 90% |

Critical path modules (Plugin Registry, Plugin Lifecycle Manager, Core Event Bus, Dependency Injection Container) must individually maintain 95% across all three dimensions.

Coverage reports are generated as part of the CI pipeline and are published as build artefacts. Coverage that falls below the minimum causes the CI pipeline to fail and blocks merging.

For further detail on CI configuration, see `testing/02-coverage-requirements.md`.

---

## What Coverage Does Not Guarantee

Coverage is a necessary but not sufficient measure of test quality. 100% line coverage does not mean the tests are meaningful.

The following are indications of coverage-gaming that must be flagged in code review:
- Tests with no assertions (they execute code but verify nothing).
- Tests that only verify the happy path, leaving all error and edge case branches uncovered.
- Tests that assert on mocked object interactions rather than observable outcomes.

High-quality tests verify observable, meaningful outcomes. Coverage is one proxy for completeness, not a substitute for thoughtful test design.

---

## References

- `ADR-0003` — Testing Standards and Coverage Requirements decision
- `testing/02-coverage-requirements.md` — CI configuration and tooling
- `testing/03-testing-patterns.md` — Test double strategies, fake implementations
- `architecture/01-plugin-system.md` — Plugin Contract (basis for Tier 3 contract tests)
- `principles/01-solid.md` — DIP enables the testability model described here
