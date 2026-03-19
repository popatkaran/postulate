# Extensibility Architecture

**Document Version:** 1.0  
**Last Updated:** 2025-01-01  
**Owner:** Architecture Review Board  
**Status:** Approved

---

## Purpose

This document defines how the system is designed to be extended over time: how extension points are declared, how new capabilities are added without modifying existing code, how plugin contracts are versioned, and which design patterns are approved for implementing extensibility. This document works in conjunction with `architecture/01-plugin-system.md`, which defines the plugin runtime mechanics.

---

## Extensibility Principles

### Principle 1: Declare Extension Points Explicitly

Extension points are not ad hoc hooks. Each extension point is a named, documented, typed declaration in the core. It has:
- A unique name.
- A defined input payload type.
- A defined output or result type.
- A documented invocation guarantee (synchronous, asynchronous, ordered, unordered).
- A defined failure semantics (fail-fast, isolated, or best-effort).

Extension points are the public API of the core for plugin authors. They are subject to the same versioning discipline as any public API.

### Principle 2: The Core Does Not Know Plugin Names

The core dispatches to all registered handlers for a given extension point. It does not reference any plugin by name or type. This is the direct application of the Dependency Inversion Principle: the core depends on the abstraction (the extension point contract), not on any concrete plugin.

### Principle 3: Extensions Must Not Break Existing Behaviour

Adding a new extension point, or adding a new optional method to the Plugin Contract, must not require any change to existing, compliant plugin implementations. This is the direct application of the Open/Closed Principle.

### Principle 4: Every Extension Point Has a Defined Execution Model

There are three approved execution models for extension points. The model is chosen at declaration time and must be documented:

| Model | Description | Use Case |
|-------|-------------|---------|
| Synchronous Sequential | Plugins invoked one by one in dependency-resolved order. Each receives the result of the previous. | Transformation pipelines |
| Synchronous Parallel | Plugins invoked concurrently. Results are collected. Order is not guaranteed. | Notification fanout, audit logging |
| Asynchronous Fire-and-Forget | Plugins invoked asynchronously. The caller does not wait for completion. | Background processing, analytics |

---

## Extension Point Design Patterns

### Pattern 1: Transformer Chain (Synchronous Sequential)

Used when each plugin may modify the output that the next plugin receives. The payload is passed through a chain of plugins, each of which may transform it.

```
// Pseudo-code

// Extension point declaration
TRANSFORM_OUTPUT: ExtensionPoint<OutputPayload, OutputPayload>
    model: SYNCHRONOUS_SEQUENTIAL
    failure: FAIL_FAST

// Registry dispatch implementation for Transformer Chain
method dispatchTransformer(point: ExtensionPoint, initial: Payload): Payload
    current = initial
    for each handler in orderedHandlersFor(point):
        current = handler.handle(current)
    return current

// Example plugin implementing a transformation
class SanitisationPlugin implements Plugin:
    method onInitialise(context: PluginContext): void
        context.registerHandler(ExtensionPoints.TRANSFORM_OUTPUT, this.sanitise)

    method sanitise(payload: OutputPayload): OutputPayload
        return payload.withSanitisedFields()
```

### Pattern 2: Observer / Event Fanout (Synchronous Parallel)

Used when multiple plugins must react to an event, but their results are not dependencies of each other and the core does not consume their output.

```
// Pseudo-code

// Extension point declaration
ON_ENTITY_CREATED: ExtensionPoint<Entity, Void>
    model: SYNCHRONOUS_PARALLEL
    failure: ISOLATED  // a failure in one observer does not affect others

// Registry dispatch
method dispatchObservers(point: ExtensionPoint, payload: Payload): void
    for each handler in handlersFor(point):
        try:
            handler.handle(payload)
        catch exception:
            log.error("Observer plugin failed", handler.getPluginId(), exception)
            // continue dispatching to remaining observers
```

### Pattern 3: Async Background Processor (Asynchronous Fire-and-Forget)

Used when plugin work does not need to complete before the request response is returned. The payload is published to an internal queue; plugin handlers consume from the queue.

```
// Pseudo-code

// Extension point declaration
ON_REPORT_GENERATED: ExtensionPoint<ReportPayload, Void>
    model: ASYNCHRONOUS
    failure: RETRY_WITH_BACKOFF

// Core publishes to the async extension point
method generateReport(data: ReportData): void
    report = reportBuilder.build(data)
    reportRepository.save(report)
    extensionPointBus.publishAsync(
        ExtensionPoints.ON_REPORT_GENERATED,
        new ReportPayload(report)
    )
    // Returns immediately; plugins consume asynchronously

// Plugin handles asynchronously
class ReportDistributionPlugin implements Plugin:
    method onInitialise(context: PluginContext): void
        context.registerAsyncHandler(
            ExtensionPoints.ON_REPORT_GENERATED,
            this.distribute
        )

    method distribute(payload: ReportPayload): void
        emailService.sendReport(payload.getRecipients(), payload.getReport())
```

---

## Adding a New Extension Point

Adding an extension point is a deliberate act governed by the following process:

1. **Identify the need.** There must be a concrete requirement from an existing or planned plugin that cannot be satisfied by existing extension points.
2. **Define the extension point.** Name it, type it, choose its execution model, define its failure semantics.
3. **Document it.** Add it to the Extension Points Registry (below) and to this document.
4. **Do not add implementation until the contract is reviewed.** Extension point declarations are reviewed by the Architecture Review Board before implementation.
5. **Write the contract tests first.** Any plugin implementing the new extension point must have contract tests that verify it handles the typed payload correctly.

---

## Extension Points Registry

This table is the authoritative list of all declared extension points in the platform. It must be kept current.

| Extension Point Name | Input Type | Output Type | Execution Model | Failure Semantics |
|---------------------|------------|-------------|-----------------|-------------------|
| BEFORE_REQUEST_PROCESSED | RequestPayload | Void | Synchronous Parallel | Isolated |
| AFTER_REQUEST_PROCESSED | RequestPayload | Void | Synchronous Parallel | Isolated |
| ON_ENTITY_CREATED | Entity | Void | Synchronous Parallel | Isolated |
| ON_ENTITY_UPDATED | Entity | Void | Synchronous Parallel | Isolated |
| ON_ENTITY_DELETED | EntityId | Void | Synchronous Parallel | Isolated |
| TRANSFORM_OUTPUT | OutputPayload | OutputPayload | Synchronous Sequential | Fail-Fast |
| ON_AUTHENTICATION_SUCCESS | AuthContext | Void | Synchronous Parallel | Isolated |
| ON_AUTHENTICATION_FAILURE | AuthAttempt | Void | Synchronous Parallel | Isolated |

---

## Versioning Extension Points

Extension point versioning follows the same major/minor/patch policy as the Plugin Contract (see `architecture/01-plugin-system.md`). The following rules apply specifically to extension points:

- **Payload fields may be added** in a minor version. Existing plugins that do not reference new fields are unaffected.
- **Payload fields must not be removed or renamed** without a major version bump and a defined migration period.
- **Execution model changes are always breaking.** A synchronous extension point that becomes asynchronous requires a major version bump.
- **Failure semantics changes are always breaking.** A plugin written for ISOLATED failure semantics may not behave correctly if the semantics change to FAIL_FAST.

---

## Anti-Patterns to Avoid

The following patterns undermine extensibility and must not be used.

### Anti-Pattern 1: instanceof Checks in the Core

```
// VIOLATION — core knows about concrete plugin types
method dispatch(payload):
    for each plugin in plugins:
        if plugin instanceof AuditPlugin:      // core must never reference a plugin type
            (plugin as AuditPlugin).audit(payload)
```

The correct approach is to dispatch through the extension point contract. If a plugin needs special invocation, it is a sign the extension point model is incorrect.

### Anti-Pattern 2: Bypassing the Registry

```
// VIOLATION — direct instantiation bypasses lifecycle and isolation
method process(request):
    plugin = new EmailNotificationPlugin()  // lifecycle not managed, not isolated
    plugin.notify(request)
```

All plugin invocations must go through the Plugin Registry.

### Anti-Pattern 3: Shared Mutable Plugin State

```
// VIOLATION — two plugins share a mutable object
sharedState = new SharedCache()
pluginA = new PluginA(sharedState)
pluginB = new PluginB(sharedState)  // creates hidden coupling between plugins
```

Plugins must communicate through the domain event system, not through shared in-process state.

---

## References

- `architecture/01-plugin-system.md` — Plugin Contract, Registry, and lifecycle
- `architecture/00-overview.md` — Architectural layers
- `principles/03-design-patterns.md` — Strategy, Observer, Chain of Responsibility patterns
- `ADR-0001` — Plugin-First Architecture
- `ADR-0002` — SOLID Principles (OCP, DIP are the theoretical basis for this document)
