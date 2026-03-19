# SOLID Principles Reference

**Document Version:** 1.0  
**Last Updated:** 2025-01-01  
**Owner:** Architecture Review Board  
**Status:** Approved  
**ADR Reference:** ADR-0002

---

## Purpose

This document is the comprehensive reference for SOLID principles as applied in this project. It expands on the definitions in ADR-0002 with extended examples, edge cases, common misapplications, and guidance for applying each principle in the context of the plugin-first architecture. Every engineer must read this document before writing production code.

Note: All code examples in this document are written in language-agnostic pseudo-code. Actual syntax will vary by implementation language. Examples are illustrative, not prescriptive.

---

## S — Single Responsibility Principle

### Definition

A class should have one, and only one, reason to change. A "reason to change" corresponds to one stakeholder or concern whose requirements could drive a change to that class.

### Identifying a Violation

Ask: "If I change requirement X, does this class have to change?" If the answer is yes for more than one independent X, the class has multiple responsibilities.

Common symptoms of SRP violations:
- Class name contains "And", "Manager", "Handler", or "Processor" with a long list of operations.
- Class has more than one group of methods that call completely different collaborators.
- Class imports both a database driver and an email client.
- Unit test for the class requires setting up more than two or three unrelated collaborators.

### Extended Example

```
// VIOLATION — UserManager does too much. Three independent reasons to change:
// 1. Authentication rules change
// 2. Notification content changes
// 3. Persistence technology changes

class UserManager:
    method register(email, password):
        hash = bcrypt(password)
        db.insert("users", {email, hash})
        smtp.send(email, "Welcome!", "Hello...")
        auditLog.write("REGISTER", email, now())

    method login(email, password):
        user = db.query("SELECT * FROM users WHERE email = ?", email)
        if not bcrypt.verify(password, user.hash):
            raise InvalidCredentialsException()
        token = jwt.sign({userId: user.id})
        return token

// COMPLIANT — each class has one responsibility

class PasswordHasher:
    method hash(plaintext): HashedPassword
    method verify(plaintext, hash): boolean

class UserRepository:
    method insert(user: NewUser): void
    method findByEmail(email: Email): Optional<User>

class WelcomeEmailSender:
    method sendWelcome(email: Email): void

class AuditLogger:
    method log(action: AuditAction, subject: string): void

class UserTokenIssuer:
    method issue(userId: UserId): AuthToken

class UserRegistrationService:
    constructor(hasher, repo, emailSender, auditLogger)
    method register(email, password): void
        hash = hasher.hash(password)
        user = new NewUser(email, hash)
        repo.insert(user)
        emailSender.sendWelcome(email)
        auditLogger.log(AuditAction.REGISTER, email)

class AuthenticationService:
    constructor(repo, hasher, tokenIssuer)
    method authenticate(email, password): AuthToken
        user = repo.findByEmail(email)
            .orElseThrow(UserNotFoundException)
        if not hasher.verify(password, user.hashedPassword):
            raise InvalidCredentialsException()
        return tokenIssuer.issue(user.id)
```

### SRP Applied to Plugins

Each plugin should have one responsibility. A plugin that handles auditing and notification is two plugins. The test for a plugin's scope: can you describe its full purpose in one sentence without using "and"?

---

## O — Open/Closed Principle

### Definition

A software module should be open for extension and closed for modification. Once a module is written, tested, and deployed, new behaviour should be addable without editing that module.

### The Mechanism of OCP

OCP is achieved through the use of abstractions. Instead of modifying a class to handle a new case, a new class implementing an abstraction is added. The original class dispatches to the abstraction and does not need to know about new implementations.

### Extended Example: Payment Processing

```
// VIOLATION — every new payment method requires modifying PaymentProcessor

class PaymentProcessor:
    method charge(amount, method, details):
        if method == "stripe":
            stripeClient.charge(details.cardToken, amount)
        elif method == "paypal":
            paypalClient.createOrder(details.paypalEmail, amount)
        elif method == "bank_transfer":      // <-- new requirement: edit required
            bankClient.initiateTransfer(details.iban, amount)
        else:
            raise UnknownPaymentMethodException()

// COMPLIANT — new payment methods are added as new classes; PaymentProcessor unchanged

interface PaymentGateway:
    method charge(amount: Money, details: PaymentDetails): ChargeResult
    method getMethodIdentifier(): string

class StripeGateway implements PaymentGateway:
    method charge(amount, details): ChargeResult
        return stripeClient.charge(details.cardToken, amount)
    method getMethodIdentifier(): string = "stripe"

class PaypalGateway implements PaymentGateway: ...
class BankTransferGateway implements PaymentGateway: ...  // new: no modification elsewhere

class PaymentGatewayRegistry:
    private gateways: Map<string, PaymentGateway>
    method register(gateway: PaymentGateway): void
        gateways.put(gateway.getMethodIdentifier(), gateway)
    method resolve(method: string): PaymentGateway
        return gateways.get(method)
            .orElseThrow(UnknownPaymentMethodException)

class PaymentProcessor:
    constructor(gatewayRegistry: PaymentGatewayRegistry)
    method charge(amount, method, details):
        gateway = gatewayRegistry.resolve(method)
        return gateway.charge(amount, details)
```

The relationship between OCP and the Plugin System is direct: the Plugin Registry is the mechanism by which the core remains closed to modification while being open to the extension provided by new plugins.

---

## L — Liskov Substitution Principle

### Definition

If S is a subtype of T, then objects of type T in a program may be replaced with objects of type S without altering any of the desirable properties of that program.

In practical terms: a concrete implementation must honour the full contract of the abstraction it implements — including implicit expectations about what the method will and will not do.

### Contract Components That Must Be Honoured

- **Pre-conditions:** A subtype must not strengthen pre-conditions. If the base contract accepts any non-null string, the subtype cannot reject strings shorter than 10 characters.
- **Post-conditions:** A subtype must not weaken post-conditions. If the base contract promises to return a non-null result, the subtype cannot return null.
- **Invariants:** Class-level invariants maintained by the base type must be maintained by the subtype.
- **Exceptions:** A subtype must not raise exceptions that the base contract does not declare or imply.
- **Return types:** In languages that allow covariant return types, this is acceptable. Contravariant or unrelated return types are violations.

### Extended Example

```
// Base contract
interface FileStorage:
    // Pre-condition: path is non-null, content is non-null
    // Post-condition: file is persisted; subsequent read returns same content
    // Exception: raises StorageException on I/O failure only
    method write(path: string, content: Bytes): void

    // Post-condition: returns non-null Bytes if path exists
    // Exception: raises FileNotFoundException if path does not exist, StorageException on I/O failure
    method read(path: string): Bytes

// LSP VIOLATION — ReadOnlyStorage strengthens the pre-condition on write
// and raises an exception the contract does not declare for write()
class ReadOnlyStorage implements FileStorage:
    method write(path, content):
        raise UnsupportedOperationException()  // VIOLATION: contract says StorageException only
    method read(path): Bytes
        return localDisk.read(path)

// COMPLIANT — model the distinction at the abstraction level
interface ReadableStorage:
    method read(path: string): Bytes

interface WritableStorage extends ReadableStorage:
    method write(path: string, content: Bytes): void

class LocalDiskStorage implements WritableStorage: ...
class ArchiveStorage implements ReadableStorage: ...  // never offered write capability
```

### LSP and Plugin Implementations

Every plugin implements the Plugin Contract (a formal interface). LSP requires that any plugin implementation can be treated as a Plugin without the registry or the core needing to know its concrete type. Plugin contract tests (Tier 3 tests in ADR-0003) are the mechanism for verifying LSP compliance.

---

## I — Interface Segregation Principle

### Definition

Clients should not be forced to depend on interfaces they do not use. A class that implements an interface should not be burdened with implementing methods it has no logical reason to support.

### Detecting a Violation

A violation exists when:
- An implementing class throws "UnsupportedOperationException" or equivalent for interface methods.
- A client imports an interface but only calls one or two of its ten methods.
- Two groups of methods in an interface are never used together by the same client.

### Extended Example

```
// VIOLATION — all implementations must carry all methods
interface DataWorker:
    method processRecord(record: Record): void
    method generateSummaryReport(): Report
    method archiveOldRecords(before: Date): void
    method sendAlerts(thresholds: AlertConfig): void

// A class that only processes records must still stub out three irrelevant methods
class StreamProcessor implements DataWorker:
    method processRecord(record): void: ...  // only real implementation
    method generateSummaryReport(): Report: raise NotImplementedException()
    method archiveOldRecords(before): void: raise NotImplementedException()
    method sendAlerts(thresholds): void: raise NotImplementedException()

// COMPLIANT — segregated into role-specific interfaces
interface RecordProcessor:
    method processRecord(record: Record): void

interface Reportable:
    method generateSummaryReport(): Report

interface Archivable:
    method archiveOldRecords(before: Date): void

interface AlertSender:
    method sendAlerts(thresholds: AlertConfig): void

// Each class implements only what it needs
class StreamProcessor implements RecordProcessor:
    method processRecord(record): void: ...

class DailyReportGenerator implements RecordProcessor, Reportable:
    method processRecord(record): void: ...
    method generateSummaryReport(): Report: ...
```

### ISP Applied to the Plugin Contract

The Plugin Contract itself is kept narrow. Optional lifecycle methods (getHealth, getMetrics) have default no-op implementations in a base class. Plugins that do not expose metrics are not burdened with implementing a metrics method. As the contract evolves, new optional capabilities are introduced through mixin interfaces or default implementations, never by adding required methods to the core contract.

---

## D — Dependency Inversion Principle

### Definition

- High-level modules should not depend on low-level modules. Both should depend on abstractions.
- Abstractions should not depend on details. Details should depend on abstractions.

### The Practical Implication

Domain and Application Layer classes must depend on interfaces, not on concrete classes. Concrete classes are instantiated only at the Composition Root (see `architecture/04-layered-architecture.md`). This makes every class unit-testable in isolation by substituting real implementations with fakes or stubs.

### Extended Example

```
// VIOLATION — high-level business logic is chained to low-level infrastructure

class InvoiceService:
    constructor():
        this.db = new PostgresDatabase()        // concrete, untestable
        this.pdfRenderer = new ITextPdfLib()    // concrete, untestable
        this.emailClient = new SendgridClient() // concrete, untestable

    method generateAndSend(orderId):
        order = this.db.query(...)
        pdf = this.pdfRenderer.render(order)
        this.emailClient.send(order.customerEmail, pdf)

// COMPLIANT — all dependencies are abstractions, injected at construction

interface OrderReadRepository:
    method findById(id: OrderId): Order

interface InvoiceRenderer:
    method render(order: Order): Bytes

interface EmailDispatcher:
    method send(recipient: EmailAddress, attachment: Bytes): void

class InvoiceService:
    constructor(
        orderRepo: OrderReadRepository,
        renderer: InvoiceRenderer,
        emailDispatcher: EmailDispatcher
    )

    method generateAndSend(orderId: OrderId): void
        order = orderRepo.findById(orderId)
        pdf = renderer.render(order)
        emailDispatcher.send(order.getCustomerEmail(), pdf)

// Test can use fakes:
class InMemoryOrderRepository implements OrderReadRepository: ...
class HtmlPreviewRenderer implements InvoiceRenderer: ...
class RecordingEmailDispatcher implements EmailDispatcher: ...

test "given order, when invoice generated, then email is dispatched":
    emailSpy = new RecordingEmailDispatcher()
    service = new InvoiceService(
        new InMemoryOrderRepository([testOrder]),
        new HtmlPreviewRenderer(),
        emailSpy
    )
    service.generateAndSend(testOrder.id)
    assert emailSpy.sentCount() == 1
    assert emailSpy.lastRecipient() == testOrder.customerEmail
```

---

## SOLID Application Checklist

Use this checklist during code review to verify SOLID compliance.

| Principle | Check |
|-----------|-------|
| SRP | Does this class have exactly one reason to change? Can its full purpose be stated in one sentence without "and"? |
| OCP | Can new behaviour be added by adding a new class rather than modifying this one? Does this class use a switch/if-else on a type that will grow? |
| LSP | Do all implementations honour the full contract including implicit expectations? Are there any methods that throw "not supported"? |
| ISP | Do any implementations leave methods empty or throw "not supported"? Do any clients only use a subset of the interface? |
| DIP | Are all collaborators received through constructors typed as interfaces? Are any concrete infrastructure types instantiated inside this class? |

---

## References

- `ADR-0002` — Mandatory enforcement policy
- `principles/02-oop-concepts.md` — Supporting OOP concepts
- `principles/03-design-patterns.md` — Design patterns that implement SOLID principles
- `guides/contributing.md` — Code review checklist
