# Postulate

> Production-ready microservices, from a single command.

## The Problem

Enterprise engineering teams waste days — sometimes weeks — before a new microservice is genuinely production-ready. Engineers copy Dockerfiles, CI pipelines, and logging wrappers from existing services, each time introducing subtle drift and encoding tribal knowledge the next engineer won't have.

The result: twenty services, ten Dockerfile variants, four logging formats, no consistent error shape, no consistent resilience patterns. Every service a unique artifact. Standards drift silently. Critical concerns — circuit breakers, distributed tracing, secret rotation, egress policy, SLO definitions — are skipped not through negligence, but unawareness.

Compliance is retrofitted after incidents. Third-party integrations are discovered in production. Platform improvements require every developer to update their local tooling to take effect.

## What Postulate Is

Postulate is an **API-first, plugin-extensible platform** for governed microservice skeleton generation.

Answer a guided set of questions. Receive a complete, production-ready microservice skeleton — one that compiles, runs, passes tests, builds a container, and deploys to Kubernetes without manual intervention.

```
postulate generate example-service
```

Every skeleton carries platform standards structurally embedded. Not documented. Not recommended. Generated and enforced.

The platform team updates the generation engine once. Every subsequent generation benefits immediately, regardless of client version.

## What It Is Not

- Not a code generator for business logic — Postulate generates the skeleton, the team writes the domain
- Not a runtime tool — it generates static project structure and configuration
- Not a one-size-fits-all public scaffolding tool — it is intentionally designed for enterprise platform engineering

## Status

Early development — Phase 1 in progress.