# ADR-001: Adoption of libp2p and GossipSub for P2P Networking

## Status
Accepted

## Context
The platform initially utilized custom HTTP POST broadcasts for peer communication. While sufficient for local prototyping, custom HTTP mechanisms scale poorly ($O(N^2)$ network traffic), lack peer identity verification, and do not handle NAT traversal or structured topology routing.

## Decision
We adopt `go-libp2p` with the GossipSub publish-subscribe routing algorithm (`github.com/libp2p/go-libp2p-pubsub`) for all peer-to-peer messaging.

## Consequences
- **Pros:** Scalable peer mesh, cryptographically signed peer identities, automatic pubsub propagation, industry-standard networking engine.
- **Cons:** Increased dependency graph size and binary footprint.
