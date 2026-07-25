# ADR-002: Byte-Level Decoupled State Machine Architecture

## Status
Accepted

## Context
Tightly coupling state mutations to block headers or transaction data structures restricts the platform to blockchain-specific applications, preventing its reuse as a general-purpose distributed state machine (e.g., Raft key-value store, distributed lock manager).

## Decision
We decouple state processing into a byte-level `StateMachine` contract: `Apply(commitLog []byte) ([]byte, error)`. Consensus engines pass raw bytes to the state machine, leaving serialization and domain logic strictly inside the application layer.

## Consequences
- **Pros:** Zero consensus-layer dependencies on domain data types; allows running arbitrary state engines on top of Sovereign.
- **Cons:** Requires explicit marshaling/unmarshaling inside individual application state wrappers.
