import json
import os
import sys

def generate_plots():
    json_path = "./results/json/cluster_benchmark_results.json"
    if not os.path.exists(json_path):
        print(f"File {json_path} not found. Skipping plot generation.")
        return

    with open(json_path, 'r') as f:
        data = json.load(f)

    results = data.get("results", [])
    out_dir = "./results/plots"
    os.makedirs(out_dir, exist_ok=True)

    # 1. Output ASCII Text Summary Matrix
    summary_file = os.path.join(out_dir, "plot_data_summary.txt")
    with open(summary_file, 'w') as f:
        f.write("=== SOVEREIGN RUNTIME BENCHMARK METRICS ===\n\n")
        f.write(f"{'Nodes':<8} | {'TPS':<10} | {'P50 (ms)':<10} | {'P95 (ms)':<10} | {'P99 (ms)':<10} | {'Recovery (ms)':<12}\n")
        f.write("-" * 72 + "\n")
        for r in results:
            f.write(f"{r['cluster_size']:<8} | {r['achieved_tps']:<10.2f} | {r['p50_latency_ms']:<10.2f} | {r['p95_latency_ms']:<10.2f} | {r['p99_latency_ms']:<10.2f} | {r['recovery_time_ms']:<12.2f}\n")

    print(f"Generated text summary at {summary_file}")

    # 2. Matplotlib Visual PNG Figures
    try:
        import matplotlib.pyplot as plt
    except ImportError:
        print("Matplotlib not installed. Installed text plot summary only.")
        return

    nodes = [r['cluster_size'] for r in results]
    tps = [r['achieved_tps'] for r in results]
    p50 = [r['p50_latency_ms'] for r in results]
    p95 = [r['p95_latency_ms'] for r in results]
    p99 = [r['p99_latency_ms'] for r in results]
    recovery = [r['recovery_time_ms'] for r in results]

    plt.style.use('ggplot')

    # Figure 1: Throughput vs Node Cluster Size
    plt.figure(figsize=(8, 5))
    plt.plot(nodes, tps, marker='o', color='#1f77b4', linewidth=2.5, label='Achieved TPS')
    plt.title('Sovereign System Throughput Scaling', fontsize=12, fontweight='bold')
    plt.xlabel('Cluster Node Count', fontsize=10)
    plt.ylabel('Transactions Per Second (TPS)', fontsize=10)
    plt.grid(True, linestyle='--', alpha=0.6)
    plt.legend()
    plt.tight_layout()
    tps_img_path = os.path.join(out_dir, "throughput_vs_nodes.png")
    plt.savefig(tps_img_path, dpi=300)
    plt.close()

    # Figure 2: Latency Percentiles vs Node Cluster Size
    plt.figure(figsize=(8, 5))
    plt.plot(nodes, p50, marker='s', color='#2ca02c', linewidth=2, label='P50 Latency')
    plt.plot(nodes, p95, marker='^', color='#ff7f0e', linewidth=2, label='P95 Latency')
    plt.plot(nodes, p99, marker='d', color='#d62728', linewidth=2, label='P99 Latency')
    plt.title('Sovereign Execution Latency Percentiles', fontsize=12, fontweight='bold')
    plt.xlabel('Cluster Node Count', fontsize=10)
    plt.ylabel('Latency (ms)', fontsize=10)
    plt.grid(True, linestyle='--', alpha=0.6)
    plt.legend()
    plt.tight_layout()
    lat_img_path = os.path.join(out_dir, "latency_percentiles.png")
    plt.savefig(lat_img_path, dpi=300)
    plt.close()

    # Figure 3: Fault Recovery Duration vs Node Cluster Size
    plt.figure(figsize=(8, 5))
    plt.plot(nodes, recovery, marker='x', color='#9467bd', linewidth=2.5, label='Quorum Recovery Time')
    plt.title('Raft Consensus Quorum Recovery Time Under Partition', fontsize=12, fontweight='bold')
    plt.xlabel('Cluster Node Count', fontsize=10)
    plt.ylabel('Recovery Duration (ms)', fontsize=10)
    plt.grid(True, linestyle='--', alpha=0.6)
    plt.legend()
    plt.tight_layout()
    rec_img_path = os.path.join(out_dir, "recovery_time_under_chaos.png")
    plt.savefig(rec_img_path, dpi=300)
    plt.close()

    print(f"Successfully generated Matplotlib PNG figures in {out_dir}:")
    print(" - throughput_vs_nodes.png")
    print(" - latency_percentiles.png")
    print(" - recovery_time_under_chaos.png")

if __name__ == "__main__":
    generate_plots()
