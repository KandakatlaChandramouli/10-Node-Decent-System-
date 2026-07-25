# Sovereign Runtime Benchmark & Evaluation Methodology

## Overview
This document specifies the empirical evaluation framework used to measure throughput (TPS), latency percentiles ($P_{50}, P_{95}, P_{99}$), fault recovery times, and resource scaling across Sovereign cluster configurations (1 to 100 nodes).

## Evaluated Metrics

1. **Throughput (Transactions Per Second - TPS):** Rate of state transition operations committed by the execution router per second.
2. **Latency Percentiles ($P_{50}, P_{95}, P_{99}$):** Microsecond-precision timings recorded from payload submission to state root commitment.
3. **Recovery Latency:** Time taken by the Raft consensus engine to achieve majority quorum following a simulated node crash.
4. **Memory Footprint & CPU Scaling:** Memory allocation and CPU utilization overhead across node cluster sizes.

## Reproducibility Commands

To execute the experiment matrix and regenerate all research artifacts:

```bash
make experiments
make plots
