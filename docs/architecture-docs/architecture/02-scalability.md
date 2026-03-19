# Scalability Architecture

**Document Version:** 1.0  
**Last Updated:** 2025-01-01  
**Owner:** Architecture Review Board  
**Status:** Approved  
**ADR Reference:** ADR-0004

---

## Purpose

This document defines the scalability model for the platform: how it achieves horizontal scale, which patterns are approved for use, how state is managed without compromising scale-out, and how the system is observed in production. All architectural decisions affecting runtime performance, deployment topology, or shared resource usage must be validated against the principles in this document.

---

## Horizontal Scaling Model

The platform scales by running multiple identical, stateless instances behind a load balancer. No instance holds state that is required for processing a request that may have started on a different instance.

```
                    +----------------+
                    |  Load Balancer |
                    +-------+--------+
                            |
           +----------------+----------------+
           |                |                |
   +-------+----+   +-------+----+   +-------+----+
   |  Instance  |   |  Instance  |   |  Instance  |
   |     A      |   |     B      |   |     C      |
   +-------+----+   +-------+----+   +-------+----+
           |                |                |
           +----------------+----------------+
                            |
                 +----------+-----------+
                 |   External State     |
                 |  (Cache / DB / Queue)|
                 +----------------------+
```

This topology has the following properties:
- Adding Instance D requires no code change and no data migration.
- Removing Instance A loses no data because all state is externalised.
- Any instance can serve any request because there is no warm-up dependency on local state.

---

## State Classification and Management

All state in the system must be classified into one of the following categories. The classification determines where and how the state is stored.

### Category 1: Request-Scoped State

State that exists only for the duration of a single request. Held in local variables within the request execution context. Never shared across requests or instances. No externalisation required.

```
// Pseudo-code — request-scoped state lives in local variables

method handleRequest(request: Request): Response
    validatedInput = validate(request)      // request-scoped
    entity = repository.find(validatedInput.id) // request-scoped
    result = entity.apply(validatedInput)   // request-scoped
    return ResponseMapper.map(result)
```

### Category 2: Session State

State that spans multiple requests from the same logical user session. Must be stored in an external, distributed session store. Instance affinity (sticky sessions) must not be used.

### Category 3: Application State

State shared across all instances, such as feature flag values, plugin configuration, or system-wide counters. Must be stored in a distributed cache or database. Reads may be served from a bounded local cache with a defined TTL to reduce latency.

### Category 4: Durable Domain State

The authoritative record of business entities. Stored in the primary database. All reads and writes go through Repository abstractions.

---

## Approved Scalability Patterns

### Circuit Breaker

**Purpose:** Prevent cascading failure when an external dependency (database, external API, downstream service) becomes unavailable or slow.

**Behaviour:**
- CLOSED: Requests pass through normally. Failures are counted.
- OPEN: After a threshold of failures, the breaker opens. Subsequent calls fail fast without invoking the dependency. A fallback is invoked.
- HALF-OPEN: After a wait period, a limited number of probe requests are allowed through. If they succeed, the breaker closes. If they fail, it reopens.

```
// Pseudo-code

class CircuitBreaker:
    private state: CircuitState = CLOSED
    private failureCount: int = 0
    private lastFailureTime: Timestamp

    method call(operation: Callable, fallback: Callable): Result
        if state == OPEN:
            if elapsed(lastFailureTime) > resetTimeout:
                state = HALF_OPEN
            else:
                return fallback()

        try:
            result = operation()
            onSuccess()
            return result
        catch exception:
            onFailure()
            return fallback()

    method onSuccess():
        failureCount = 0
        state = CLOSED

    method onFailure():
        failureCount++
        lastFailureTime = now()
        if failureCount >= threshold:
            state = OPEN
```

**When to use:** Any call to an external dependency (database, cache, HTTP client, message queue) that could fail independently of the core.

### Bulkhead

**Purpose:** Prevent a slow or failing plugin or external call from exhausting shared resources (thread pools, connection pools) and thereby degrading the entire system.

**Implementation:** Each plugin execution context and each external dependency is allocated a bounded resource pool. One pool exhausting does not affect another.

```
// Pseudo-code

class BulkheadExecutor:
    private pool: BoundedExecutorPool

    constructor(maxConcurrent: int, queueSize: int)
        pool = new BoundedExecutorPool(maxConcurrent, queueSize)

    method execute(task: Callable): Future
        if pool.isAtCapacity():
            raise BulkheadCapacityExceededException()
        return pool.submit(task)
```

**When to use:** Wrapping plugin dispatch calls, external HTTP client calls, and long-running background tasks.

### Rate Limiter

**Purpose:** Protect shared downstream resources from being overwhelmed by a single consumer or a burst of traffic.

**Implementation:** Token bucket or sliding window algorithm. State (token count) must be stored externally when rate limiting must apply across multiple instances.

```
// Pseudo-code

interface RateLimiter:
    // Returns true if the request is permitted; false if the limit is exceeded.
    method tryAcquire(key: string, permitCount: int): boolean

class DistributedTokenBucketRateLimiter implements RateLimiter:
    constructor(store: AtomicCounterStore, capacity: int, refillRate: int)

    method tryAcquire(key: string, permitCount: int): boolean
        tokens = store.get(key)
        if tokens >= permitCount:
            store.decrement(key, permitCount)
            return true
        return false
```

### Read-Through Cache

**Purpose:** Reduce latency for frequently read, rarely changed data (configuration, reference data, entity lookups).

**Implementation:** The cache sits in front of the primary repository. On a cache miss, the value is fetched from the repository and written into the cache with a defined TTL.

```
// Pseudo-code

class CachedRepository implements Repository:
    constructor(delegate: Repository, cache: DistributedCache, ttl: Duration)

    method findById(id: EntityId): Entity
        cached = cache.get(id)
        if cached is present:
            return cached

        entity = delegate.findById(id)
        cache.set(id, entity, ttl)
        return entity

    method save(entity: Entity): void
        delegate.save(entity)
        cache.invalidate(entity.getId())  // invalidate on write
```

**Cache TTL policy:** TTL values must be defined per entity type in configuration, not hardcoded. Default TTL is 60 seconds unless documented otherwise.

### Asynchronous Event Processing

**Purpose:** Decouple producers of domain events from consumers. Enable workloads that do not need to be completed within the request-response cycle to be processed asynchronously.

**Pattern:** Domain events raised by the core are published to a durable message queue. Event consumers (which may be plugins) subscribe to specific event types and process them independently.

```
// Pseudo-code

// Producer side (Application Layer)
method handleCommand(command: CreateOrderCommand): void
    order = Order.create(command)
    orderRepository.save(order)
    eventBus.publish(new OrderCreatedEvent(order.getId()))

// Consumer side (async, may be a plugin or an internal handler)
class OrderNotificationPlugin implements Plugin:
    method onInitialise(context: PluginContext): void
        context.registerHandler(ExtensionPoints.ON_ENTITY_CREATED, this.handleOrderCreated)

    method handleOrderCreated(event: OrderCreatedEvent): void
        notificationService.notifyCustomer(event.getOrderId())
```

---

## Observability Requirements

All instances must emit the following signals to enable operational visibility.

### Structured Logging

All log entries must be structured (e.g., JSON format) and must include the following fields as a minimum:

| Field | Description |
|-------|-------------|
| timestamp | ISO 8601 UTC |
| level | DEBUG, INFO, WARN, ERROR |
| traceId | Distributed trace identifier, propagated from the incoming request |
| spanId | Current span identifier |
| serviceId | Logical service name |
| instanceId | Unique identifier of the current instance |
| message | Human-readable description |

No log entry may contain sensitive personal data (passwords, authentication tokens, PII) in any field.

### Metrics

The following metrics must be emitted by the core. Plugins must emit equivalent metrics for their own operations.

| Metric Name | Type | Description |
|-------------|------|-------------|
| request.count | Counter | Total requests received |
| request.duration | Histogram | Request processing time in milliseconds |
| request.error.count | Counter | Total requests resulting in an error |
| plugin.dispatch.count | Counter | Total extension point dispatch calls |
| plugin.dispatch.failure.count | Counter | Total plugin dispatch calls that resulted in a caught exception |
| plugin.dispatch.duration | Histogram | Time spent in plugin dispatch per extension point |
| circuitbreaker.state | Gauge | Current circuit breaker state (0=CLOSED, 1=OPEN, 2=HALF_OPEN) |

### Health Checks

Every instance must expose a health endpoint that returns the aggregate health of the instance, including the health reported by each ACTIVE plugin.

---

## References

- `ADR-0004` — Scalability and Stateless Core Design decision
- `ADR-0001` — Plugin-First Architecture (plugins must comply with stateless rules)
- `architecture/00-overview.md` — Layer model
- `architecture/01-plugin-system.md` — Plugin isolation and bulkhead integration
