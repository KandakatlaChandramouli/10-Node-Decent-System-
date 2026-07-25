# ADR-003: Multi-Application Runtime Hosting Engine

## Status
Accepted

## Context
A major architectural anti-pattern in distributed systems design is coupling runtime orchestration strictly to one domain application (e.g., a blockchain). To prove platform versatility, the runtime must host distinct state machines (e.g., Blockchains, Key-Value Stores, Distributed Lock Managers) using the exact same consensus, storage, and networking interfaces.

## Decision
We establish `applications/kv-store/` as a second first-class application alongside `applications/sovereign-chain/`. Both applications implement the `interfaces.StateMachine` contract without altering core runtime code.

## Consequences
- **Pros:** Validates platform reusability and clean interface separation.
- **Cons:** Requires explicit payload router or configuration switches when swapping application state machines.
