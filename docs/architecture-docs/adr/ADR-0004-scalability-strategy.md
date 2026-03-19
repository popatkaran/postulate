# ADR-0004: Scalability and Stateless Core Design

**Status:** Accepted  
**Date:** 2025-01-01  
**Deciders:** Architecture Review Board  
**Ticket / RFC Reference:** RFC-004

---

## Context

The system must be capable of horizontal scaling to meet demand spikes without architectural changes. Horizontal scaling — running multiple identical instances of the service — is only possible if instances do not share in-process mutable state. Any state that must persist across requests must be externalised to a dedicated state store.

The plugin-first model (ADR-0001) introduces an additional concern: if plugins themselves hold process-local state, scaling is compromised. Plugin design must therefore conform to the same stateless or externalised-state constraint as the core.

## Decision Drivers

- Any instance must be able to serve any request without warm-up or instance affinity.
- The system must scale from one to N instances by changing instance count, not by code change.
- Plugin authors must have clear guidelines that prevent them from introducing scaling bottlenecks.
- Failure of one instance must not corrupt shared state for other instances.

## Decision

**The core runtime and all plugins must be designed as stateless processing units. All persistent and shared state must be externalised to purpose-built state stores. No in-process global mutable state is permitted in core or plugin code.**

---

## Stateless Design Rules

### Rule 1: No Process-Local Shared State

In-process global variables, singletons holding mutable data, and class-level mutable fields that survive across request boundaries are prohibited. Configuration values and immutable registries are exempt.

```
// VIOLATION — singleton holds mutable state that differs between instances
class RequestCounter:
    static count = 0
    method increment(): count++
    method get(): return count

// COMPLIANT — delegate counters to an external store
interface CounterStore:
    method increment(key: string): void
    method get(key: string): long

class RequestCounter:
    constructor(store: CounterStore)
    method increment(): store.increment("requests")
```

### Rule 2: Request Context Passed Explicitly

All data required to process a request must either arrive with the request or be fetched from an external store within that request's execution context. State must not bleed between requests.

### Rule 3: Idempotent Operations Where Possible

Operations that mutate shared state must be designed to be safely retried without unintended side effects. Idempotency keys must be used for write operations that interact with external stores.

### Rule 4: Plugin State Externalisation

If a plugin requires state that persists beyond a single invocation, it must declare a dependency on an injected state store abstraction. Plugins must not use process-local caches unless those caches are read-only, bounded, and populated from an external source.

---

## Scalability Patterns Approved for Use

The following patterns are approved and documented in `architecture/02-scalability.md`. Teams must select from this list before proposing alternatives.

| Pattern | Use Case |
|---------|---------|
| Horizontal Pod Autoscaling | Stateless service layer scale-out |
| External Cache (e.g., distributed cache) | Shared, low-latency read state |
| Event-Driven Fanout | Decoupled async processing |
| Circuit Breaker | Fault isolation for external dependencies |
| Bulkhead | Resource isolation between plugin execution contexts |
| Rate Limiter | Protection of shared downstream resources |

---

## Consequences

### Positive

- Any instance can handle any request: true horizontal scalability.
- Instances can be terminated and replaced without data loss.
- Load balancing is trivially round-robin; no session affinity required.

### Negative

- Externalising state introduces a network call where an in-process lookup would be faster; this is an accepted trade-off.
- Stateless design is a harder constraint for plugin authors to work within; documentation and enforcement are necessary.

### Neutral / Follow-up Actions

- Plugin development guide must include a section on state management restrictions.
- Architecture review must include a stateless compliance check for every new plugin.
- Define the approved external state store abstractions in `architecture/02-scalability.md`.

## Compliance

- SOLID DIP: the state store dependency is always injected, never instantiated inside a plugin or core module.
- No singleton or static mutable field pattern is permitted without explicit Architecture Review Board approval.

## References

- `architecture/02-scalability.md` — Scalability patterns and deployment topology
- `ADR-0001` — Plugin-First Architecture
- `ADR-0002` — SOLID Principles Enforcement
