import json
import os

def generate_text_plots():
    json_path = "./results/json/cluster_benchmark_results.json"
    if not os.path.exists(json_path):
        print(f"File {json_path} not found. Skipping plot generation.")
        return

    with open(json_path, 'r') as f:
        data = json.load(f)

    results = data.get("results", [])
    out_dir = "./results/plots"
    os.makedirs(out_dir, exist_ok=True)

    summary_file = os.path.join(out_dir, "plot_data_summary.txt")
    with open(summary_file, 'w') as f:
        f.write("=== SOVEREIGN RUNTIME BENCHMARK METRICS ===\n\n")
        f.write(f"{'Nodes':<8} | {'TPS':<10} | {'P50 (ms)':<10} | {'P95 (ms)':<10} | {'P99 (ms)':<10} | {'Recovery (ms)':<12}\n")
        f.write("-" * 72 + "\n")
        for r in results:
            f.write(f"{r['cluster_size']:<8} | {r['achieved_tps']:<10.2f} | {r['p50_latency_ms']:<10.2f} | {r['p95_latency_ms']:<10.2f} | {r['p99_latency_ms']:<10.2f} | {r['recovery_time_ms']:<12.2f}\n")

    print(f"Generated benchmark summary visualization data at {summary_file}")

if __name__ == "__main__":
    generate_text_plots()
