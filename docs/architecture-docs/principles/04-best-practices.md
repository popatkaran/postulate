# Engineering Best Practices

**Document Version:** 1.0  
**Last Updated:** 2025-01-01  
**Owner:** Architecture Review Board  
**Status:** Approved

---

## Purpose

This document defines engineering best practices that apply to all production code in this project. These practices complement the SOLID principles (ADR-0002) and OOP concepts (`principles/02-oop-concepts.md`) and address practical day-to-day coding concerns: error handling, naming, code structure, defensive programming, and clean code standards.

All examples are written in language-agnostic pseudo-code unless otherwise noted.

---

## 1. Error Handling

### Rule 1.1: Use Typed, Meaningful Exceptions

Exceptions must carry sufficient information for a caller to understand what went wrong and for an operator to diagnose it from logs. Generic exceptions (Exception, RuntimeError, Error) must not be thrown directly; they must be extended into domain-specific types.

```
// POOR — generic exception loses diagnostic information
method findOrder(id):
    if order is null: raise Exception("not found")

// GOOD — typed exception with context
class OrderNotFoundException extends DomainException:
    constructor(orderId: OrderId)
        super("Order not found: " + orderId.value)
        this.orderId = orderId

method findOrder(id: OrderId): Order
    order = repository.findById(id)
    if order is absent: raise OrderNotFoundException(id)
    return order
```

### Rule 1.2: Fail Fast

Validate inputs at the entry point of every public method. Do not allow invalid data to flow deeper into the system where it will cause a harder-to-diagnose failure. Pre-condition violations are programming errors; they should raise immediately.

```
method transfer(fromId: AccountId, toId: AccountId, amount: Money): void
    // Fail fast on invalid inputs
    require(fromId is not null, "fromId must not be null")
    require(toId is not null, "toId must not be null")
    require(amount is not null, "amount must not be null")
    require(amount.isPositive(), "transfer amount must be positive")
    require(fromId != toId, "source and destination accounts must differ")

    // Proceed with valid inputs
    ...
```

### Rule 1.3: Never Swallow Exceptions Silently

A catch block that suppresses an exception without logging or re-raising is a debugging time-bomb. Every exception that is caught must be logged with full context, or re-raised as a different (typically higher-level) exception.

```
// VIOLATION — exception silently swallowed
try:
    cache.set(key, value)
catch exception:
    pass  // failure is invisible; caller thinks cache was written

// COMPLIANT — failure logged; system degrades gracefully
try:
    cache.set(key, value)
catch CacheException as e:
    log.warn("Cache write failed; continuing without cache", key, e)
    // The caller proceeds without the cache benefit; the failure is visible in logs
```

### Rule 1.4: Do Not Use Exceptions for Flow Control

Exceptions are for exceptional conditions — situations that should not occur in normal operation. Using exceptions as a control flow mechanism (like a GOTO for successful paths) is a performance anti-pattern and makes code difficult to read.

```
// VIOLATION — exception used for normal flow control
method getUserByEmail(email):
    try:
        return userRepo.findByEmail(email)
    catch UserNotFoundException:
        return createDefaultUser()  // not exceptional; a normal alternative path

// COMPLIANT — use Optional or explicit null check for normal absence
method getUserByEmail(email):
    return userRepo.findByEmail(email)  // returns Optional<User>
        .orElseGet(() -> createDefaultUser())
```

---

## 2. Naming

Names are the primary form of documentation. A well-named class, method, or variable eliminates the need for most comments.

### Rule 2.1: Names Must Reveal Intent

A name should answer: why does this exist, what does it do, and how is it used?

| Category | Standard | Example |
|----------|----------|---------|
| Class | Noun or noun phrase | `OrderValidator`, `PaymentGateway`, `InvoiceRenderer` |
| Interface | Noun (role) or adjective (capability) | `Serialisable`, `OrderRepository`, `EventPublisher` |
| Method | Verb or verb phrase | `calculateTax()`, `findByEmail()`, `publishEvent()` |
| Boolean method | Is/has/can prefix | `isActive()`, `hasPermission()`, `canRefund()` |
| Variable | Noun; avoid single letters except loop indices | `customerEmail`, `totalAmount`, `pluginId` |
| Constant | Upper snake case | `MAX_RETRY_COUNT`, `DEFAULT_TIMEOUT_SECONDS` |

### Rule 2.2: Avoid Misleading Names

A name that is close to but not exactly the correct meaning is worse than a less precise but accurate name. Do not name a list "accountList" — name it "accounts". Do not name a flag "flag" — name it "isPaymentOverdue".

### Rule 2.3: Searchable Names

Avoid single-character names and abbreviations for anything other than short-lived local loop variables. `customerId` is searchable; `cid` is not.

---

## 3. Function and Method Design

### Rule 3.1: Methods Do One Thing

A method should do one thing, do it well, and do it only. If you need the word "and" to describe what a method does, it does too much.

### Rule 3.2: Limit Method Length

A method that requires scrolling to read is a sign of too many responsibilities. Methods should rarely exceed 20 lines of meaningful code. If a method is growing long, extract sub-operations into named private methods. Named private methods are better documentation than inline comments.

```
// LONG and hard to follow
method processOrder(order):
    // validate
    if order.customerId is null: raise ...
    if order.lines.isEmpty(): raise ...
    for each line in order.lines:
        if line.quantity <= 0: raise ...
    // price
    subtotal = 0
    for each line in order.lines:
        price = pricingService.getPrice(line.productId)
        subtotal += price * line.quantity
    tax = taxService.calculate(subtotal, order.customer.region)
    total = subtotal + tax
    // persist and notify
    order.setTotal(total)
    orderRepo.save(order)
    emailService.sendConfirmation(order)

// REFACTORED — each step has a name
method processOrder(order: DraftOrder): ConfirmedOrder
    validateOrder(order)
    pricedOrder = applyPricing(order)
    confirmedOrder = pricedOrder.confirm()
    orderRepo.save(confirmedOrder)
    notifyCustomer(confirmedOrder)
    return confirmedOrder
```

### Rule 3.3: Limit Parameters

A method with more than three or four parameters is a sign of a missing abstraction. Introduce a parameter object.

```
// VIOLATION — seven parameters; order is unclear; easy to pass arguments in wrong positions
method createUser(firstName, lastName, email, password, role, locale, timezone): User

// COMPLIANT — parameter object
class CreateUserRequest:
    readonly firstName: string
    readonly lastName: string
    readonly email: EmailAddress
    readonly password: PlainTextPassword
    readonly role: UserRole
    readonly locale: Locale
    readonly timezone: TimeZone

method createUser(request: CreateUserRequest): User
```

---

## 4. Code Structure and Organisation

### Rule 4.1: Stepdown Rule

Code should read like a narrative, top to bottom. High-level functions appear at the top; low-level detail functions appear below. A reader can scan the high-level flow and drill into detail only when needed.

### Rule 4.2: Package / Module by Feature, Not by Layer

Within the codebase, organise code by domain feature (e.g., `billing`, `inventory`, `identity`) rather than by technical layer (e.g., `controllers`, `services`, `repositories`). Layer boundaries are enforced by dependency rules, not by directory placement.

```
// PREFERRED structure — grouped by feature
src/
  billing/
    domain/
      Invoice.class
      InvoiceRepository.interface
    application/
      GenerateInvoiceHandler.class
    infrastructure/
      SqlInvoiceRepository.class
    delivery/
      InvoiceController.class
  inventory/
    ...
  identity/
    ...

// AVOID — grouped by layer; creates artificial coupling between unrelated features
src/
  controllers/
    InvoiceController.class
    InventoryController.class
  services/
    BillingService.class
    InventoryService.class
  repositories/
    InvoiceRepository.class
    InventoryRepository.class
```

### Rule 4.3: Configuration Must Be Externalised

No magic values — timeouts, retry counts, page sizes, limits — may be hardcoded in business logic. All configurable values must be defined in a configuration object, loaded at startup, validated at startup (fail fast if required config is missing), and injected as a dependency.

---

## 5. Defensive Programming

### Rule 5.1: Validate All External Inputs

Any data entering the system from an external source (HTTP request body, message queue payload, file import, CLI argument) must be validated before it is passed to the Application or Domain Layer. The Delivery Layer is responsible for format and presence validation. The Domain Layer is responsible for business rule validation.

### Rule 5.2: Return Defensive Copies of Mutable Objects

When a method returns an internal collection or mutable object, return a copy, not a reference to the internal state. This prevents callers from inadvertently mutating object internals.

```
class PipelineConfig:
    private stages: List<Stage>

    // VIOLATION — caller can mutate internal list
    method getStages(): List<Stage>
        return stages

    // COMPLIANT — defensive copy
    method getStages(): List<Stage>
        return List.immutableCopy(stages)
```

### Rule 5.3: Null Safety

Null references are a frequent source of runtime failures. Apply the following rules:

- Methods must not return null where the caller would need a null check. Use Optional (or equivalent) for values that may be absent.
- Constructor parameters must be validated as non-null at construction time if null is not a valid state.
- Null must not be passed as a method argument. Use Optional, a null object, or a default value.

---

## 6. Logging Standards

### Rule 6.1: Log Structured Data, Not Concatenated Strings

Structured log entries (key-value pairs) are machine-parseable and enable log aggregation, search, and alerting. String concatenation produces entries that are human-readable only.

```
// POOR — unstructured, unsearchable
log.info("User " + userId + " placed order " + orderId + " for " + amount)

// GOOD — structured; fields are queryable in log aggregation tools
log.info("Order placed",
    "userId", userId,
    "orderId", orderId,
    "amount", amount.toString(),
    "currency", amount.currency
)
```

### Rule 6.2: Log Levels Must Be Used Correctly

| Level | Use For |
|-------|---------|
| DEBUG | Detailed diagnostic information, useful only during development and troubleshooting. Never enabled in production by default. |
| INFO | Normal operational events: application started, request received, background job completed. |
| WARN | Abnormal but handled conditions: cache miss, retry attempt, deprecated API call. |
| ERROR | Failure requiring investigation: exception caught, external service returned an error, business rule violated in a critical path. |

### Rule 6.3: Never Log Sensitive Data

Passwords, authentication tokens, full credit card numbers, and personally identifiable information (PII) must never appear in log output.

---

## 7. Commenting Standards

### Rule 7.1: Code Should Be Self-Documenting

The goal of naming and structure rules is to make code readable without comments. A comment that explains *what* the code does is a sign the code should be renamed or restructured.

### Rule 7.2: Comments Explain Why, Not What

Comments are for explaining intent, rationale, or constraints that cannot be expressed in code: business rules, algorithmic choices, known limitations, or context that would otherwise be lost.

```
// POOR — explains the obvious
// Multiply amount by tax rate
tax = amount * taxRate

// GOOD — explains non-obvious business rule
// VAT is applied before the loyalty discount per §4.2 of the pricing agreement
// with the EU reseller. Changing this order will break the reseller margin calculation.
taxedAmount = amount.applyVat(vatRate)
finalAmount = taxedAmount.applyLoyaltyDiscount(customer.discountRate)
```

### Rule 7.3: Public API Documentation

All public interfaces, their methods, and their parameters must be documented. The documentation must describe: the contract (what the method guarantees), pre-conditions (what the caller must ensure), and exceptions that may be raised.

---

## References

- `principles/01-solid.md` — SOLID principles
- `principles/02-oop-concepts.md` — OOP concepts
- `principles/03-design-patterns.md` — Design patterns
- `testing/01-testing-standards.md` — Testing standards
- `guides/contributing.md` — Code review checklist integrating all of the above
