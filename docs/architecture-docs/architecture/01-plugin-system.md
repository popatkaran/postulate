# Plugin System Architecture

**Document Version:** 1.0  
**Last Updated:** 2025-01-01  
**Owner:** Architecture Review Board  
**Status:** Approved  
**ADR Reference:** ADR-0001

---

## Purpose

This document defines the complete design of the plugin system: the Plugin Contract, the Plugin Registry, the lifecycle state machine, extension point mechanics, isolation requirements, and versioning policy. Every developer writing a plugin or touching plugin infrastructure must read this document before writing any code.

---

## Core Concepts

### Plugin

A Plugin is a self-contained unit of behaviour that implements the Plugin Contract. It extends system capabilities without modifying the core. A plugin:

- Has a unique identifier.
- Declares the version of the Plugin Contract it implements.
- Declares its dependencies on other plugins or services.
- Implements one or more extension points.
- Manages its own internal state (if any) within the constraints of ADR-0004.

### Plugin Contract

The Plugin Contract is the formal interface that all plugins must implement. It is the sole point of coupling between the core and any plugin. The contract is versioned independently of the platform version.

### Plugin Registry

The Plugin Registry is the central runtime catalogue of all registered plugins. It is the only mechanism through which the core invokes plugin behaviour. The core never holds a direct reference to a concrete plugin implementation.

### Extension Point

An Extension Point is a named, typed hook in the core at which plugins can inject behaviour. Extension points are declared by the core; plugins declare which extension points they wish to handle.

---

## Plugin Contract Definition

The following is the canonical Plugin Contract, expressed in language-agnostic pseudo-code. All plugins must implement this interface completely. Methods marked as optional have default no-op implementations in the base class (if the language supports it); plugins may override them.

```
// Pseudo-code — language-agnostic representation of the Plugin Contract

interface Plugin:

    // Required: returns the unique, immutable identifier for this plugin.
    // Format: reverse-domain notation, e.g. "com.example.my-plugin"
    method getId(): PluginId

    // Required: returns the semantic version of this plugin implementation.
    method getVersion(): SemanticVersion

    // Required: returns the version of the Plugin Contract this plugin implements.
    // Used by the Registry to verify compatibility before registration.
    method getContractVersion(): SemanticVersion

    // Required: returns the list of PluginIds this plugin depends on.
    // The Registry will ensure dependencies are started before this plugin.
    method getDependencies(): List<PluginId>

    // Lifecycle — Called by the Registry during the INITIALIZING phase.
    // The plugin receives its resolved dependencies and configuration here.
    // Raises PluginInitialisationException on failure.
    method onInitialise(context: PluginContext): void

    // Lifecycle — Called when the plugin transitions to ACTIVE.
    // The plugin may begin processing from this point.
    method onStart(): void

    // Lifecycle — Called when the plugin transitions to STOPPING.
    // The plugin must complete in-flight work and release acquired resources.
    // Must complete within the configured shutdown timeout.
    method onStop(): void

    // Optional: Returns health status for observability integration.
    // Default: returns HealthStatus.HEALTHY
    method getHealth(): HealthStatus

    // Optional: Returns a map of named metrics for the observability system.
    // Default: returns empty map
    method getMetrics(): Map<string, Metric>
```

---

## Plugin Context

The `PluginContext` object is provided to each plugin during initialisation. It is the only mechanism through which a plugin accesses platform services. This enforces the Dependency Inversion Principle and prevents plugins from reaching into core internals.

```
// Pseudo-code

interface PluginContext:

    // Returns the resolved configuration for this plugin.
    method getConfig(): PluginConfig

    // Returns a logger scoped to this plugin's identifier.
    method getLogger(): Logger

    // Returns an event publisher for emitting domain events.
    method getEventPublisher(): EventPublisher

    // Returns the resolved dependency plugin by id.
    // Raises DependencyNotFoundException if the dependency is not registered.
    method getDependency(id: PluginId): Plugin

    // Returns the state store for this plugin's exclusive use.
    // Backed by the external state store configured at platform level.
    method getStateStore(): KeyValueStore
```

---

## Plugin Lifecycle State Machine

Every plugin passes through a defined sequence of states. Transitions are managed exclusively by the Plugin Registry. No external code may directly manipulate a plugin's state.

```
States:
  REGISTERED    — Plugin has been submitted to the Registry. No initialisation has occurred.
  INITIALIZING  — onInitialise() is executing. Plugin is not yet available for dispatch.
  ACTIVE        — onStart() has completed. Plugin is available for extension point dispatch.
  STOPPING      — onStop() is executing. Plugin is no longer available for dispatch.
  STOPPED       — onStop() has completed. Plugin may be re-initialised or unregistered.
  FAILED        — An unhandled exception occurred during a lifecycle transition.
                  The plugin is quarantined. Manual intervention is required.

Transitions:
  REGISTERED    -> INITIALIZING  : Registry calls onInitialise()
  INITIALIZING  -> ACTIVE        : onInitialise() completes without exception
  INITIALIZING  -> FAILED        : onInitialise() raises exception
  ACTIVE        -> STOPPING      : Registry calls onStop()
  STOPPING      -> STOPPED       : onStop() completes without exception
  STOPPING      -> FAILED        : onStop() raises exception or exceeds shutdown timeout
  STOPPED       -> INITIALIZING  : Registry calls onInitialise() again (hot reload)
  FAILED        -> REGISTERED    : Manual recovery action via Registry admin API
```

Illustration:

```
REGISTERED
    |
    v
INITIALIZING ----[exception]----> FAILED
    |
    v
ACTIVE
    |
    v
STOPPING ----[exception or timeout]----> FAILED
    |
    v
STOPPED
    |
    v  (hot reload path)
INITIALIZING
```

---

## Plugin Registry

The Plugin Registry is the authoritative runtime index of plugins. It is the sole object through which the core dispatches to plugins.

### Responsibilities

- Accept plugin registrations and validate contract version compatibility.
- Resolve and validate the dependency graph before initialisation.
- Execute lifecycle transitions in dependency-correct order.
- Dispatch extension point invocations to all registered, ACTIVE plugins that handle the given extension point.
- Isolate plugin failures: a FAILED plugin must not prevent dispatch to other plugins.
- Expose health and metric aggregation for all registered plugins.

### Registration Pseudo-code

```
// Pseudo-code

class PluginRegistry:

    private plugins: Map<PluginId, PluginEntry>
    private contractVersion: SemanticVersion

    method register(plugin: Plugin): void

        // 1. Validate contract compatibility
        if not isCompatible(plugin.getContractVersion(), this.contractVersion):
            raise IncompatibleContractVersionException(plugin.getId(), plugin.getContractVersion())

        // 2. Check for duplicate registration
        if plugins.contains(plugin.getId()):
            raise DuplicatePluginException(plugin.getId())

        // 3. Record in registry with REGISTERED state
        entry = new PluginEntry(plugin, PluginStatus.REGISTERED)
        plugins.put(plugin.getId(), entry)

    method startAll(): void

        // 1. Resolve dependency order (topological sort)
        orderedIds = resolveDependencyOrder()

        // 2. Initialise each plugin in order
        for each id in orderedIds:
            entry = plugins.get(id)
            transition(entry, PluginStatus.INITIALIZING)
            try:
                context = buildContext(entry)
                entry.plugin.onInitialise(context)
                entry.plugin.onStart()
                transition(entry, PluginStatus.ACTIVE)
            catch exception:
                transition(entry, PluginStatus.FAILED)
                log.error("Plugin failed to start", id, exception)
                // Continue initialising remaining plugins

    method dispatch(extensionPoint: ExtensionPoint, payload: Payload): List<Result>

        results = []
        for each entry in activePluginsFor(extensionPoint):
            try:
                result = entry.plugin.handle(extensionPoint, payload)
                results.add(result)
            catch exception:
                // Isolate: log the failure, do not propagate to caller
                log.error("Plugin raised exception during dispatch", entry.plugin.getId(), exception)
                metrics.increment("plugin.dispatch.failure", entry.plugin.getId())

        return results
```

---

## Extension Points

Extension points are declared in the core as named constants with an associated payload type and result type. The core dispatches to all ACTIVE plugins that register a handler for the given extension point.

### Declaring an Extension Point

```
// Pseudo-code

// Core declares extension points as typed constants
namespace ExtensionPoints:
    BEFORE_REQUEST_PROCESSED: ExtensionPoint<RequestPayload, Void>
    AFTER_REQUEST_PROCESSED:  ExtensionPoint<RequestPayload, Void>
    ON_ENTITY_CREATED:        ExtensionPoint<Entity, Void>
    ON_ENTITY_DELETED:        ExtensionPoint<EntityId, Void>
    TRANSFORM_OUTPUT:         ExtensionPoint<OutputPayload, OutputPayload>
```

### Handling an Extension Point in a Plugin

```
// Pseudo-code — a plugin registers handlers during onInitialise()

class AuditPlugin implements Plugin:

    method onInitialise(context: PluginContext):
        context.registerHandler(
            ExtensionPoints.AFTER_REQUEST_PROCESSED,
            this.auditRequest
        )

    method auditRequest(payload: RequestPayload): void
        auditLog.record(payload.userId, payload.action, now())
```

---

## Isolation Requirements

Plugin failures must not affect core stability or other plugins.

1. Plugin invocations during dispatch must be executed within a fault boundary (try-catch at the Registry level).
2. Plugins must not be given direct references to core internals; they must use only what is provided through `PluginContext`.
3. If a language runtime supports it, plugin class loading should be isolated to prevent classpath pollution.
4. Long-running plugin operations should be executed with a configurable timeout. Timeout causes the plugin to transition to FAILED.

---

## Plugin Versioning and Compatibility Policy

| Contract Version Change | Compatibility Rule |
|--------------------------|-------------------|
| Patch (x.y.Z) | Fully backward compatible. No plugin changes required. |
| Minor (x.Y.0) | Backward compatible additions only. Existing plugins continue to function. New optional methods added with default no-op implementations. |
| Major (X.0.0) | Breaking change. A Compatibility Adapter must be provided that wraps old-contract plugins for a defined deprecation period (minimum two minor platform releases). |

---

## References

- `ADR-0001` — Plugin-First Architecture decision
- `ADR-0002` — SOLID Principles (OCP, DIP, LSP are critical for plugin design)
- `ADR-0003` — Testing Standards (Contract Tests apply directly to all Plugin implementations)
- `architecture/03-extensibility.md` — Extension point design patterns
- `testing/01-testing-standards.md` — Tier 3 Contract Tests
