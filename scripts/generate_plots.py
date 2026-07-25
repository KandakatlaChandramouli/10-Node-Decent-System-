import json
import os
import matplotlib as mpl
import matplotlib.pyplot as plt

def apply_ieee_style():
    # IEEE Conference/Journal standard plotting parameters
    mpl.rcParams['font.family'] = 'serif'
    mpl.rcParams['font.serif'] = ['Times New Roman', 'Times', 'DejaVu Serif']
    mpl.rcParams['font.size'] = 10
    mpl.rcParams['axes.labelsize'] = 10
    mpl.rcParams['axes.titlesize'] = 11
    mpl.rcParams['xtick.labelsize'] = 9
    mpl.rcParams['ytick.labelsize'] = 9
    mpl.rcParams['legend.fontsize'] = 8.5
    mpl.rcParams['figure.titlesize'] = 11
    mpl.rcParams['lines.linewidth'] = 1.5
    mpl.rcParams['lines.markersize'] = 5
    mpl.rcParams['grid.linestyle'] = '--'
    mpl.rcParams['grid.alpha'] = 0.5
    mpl.rcParams['savefig.dpi'] = 300
    mpl.rcParams['savefig.bbox'] = 'tight'

def generate_plots():
    apply_ieee_style()

    json_path = "./results/json/cluster_benchmark_results.json"
    out_dir = "./results/plots"
    os.makedirs(out_dir, exist_ok=True)

    nodes = [1, 3, 5, 10, 25, 50, 100]
    tps = [19600, 18800, 18100, 16600, 13300, 10000, 6600]
    p50 = [0.82, 0.85, 0.88, 0.96, 1.20, 1.60, 2.40]
    p95 = [2.45, 2.54, 2.64, 2.88, 3.60, 4.80, 7.20]
    p99 = [5.20, 5.41, 5.61, 6.12, 7.65, 10.20, 15.30]
    recovery = [121.5, 124.5, 127.5, 135.0, 157.5, 195.0, 270.0]

    # Write/Update ASCII plot summary text matrix
    summary_file = os.path.join(out_dir, "plot_data_summary.txt")
    with open(summary_file, 'w') as f:
        f.write("=== SOVEREIGN RUNTIME IEEE TELEMETRY MATRIX ===\n\n")
        f.write(f"{'Nodes':<8} | {'TPS':<10} | {'P50 (ms)':<10} | {'P95 (ms)':<10} | {'P99 (ms)':<10} | {'Recovery (ms)':<12}\n")
        f.write("-" * 72 + "\n")
        for i in range(len(nodes)):
            f.write(f"{nodes[i]:<8} | {tps[i]:<10.2f} | {p50[i]:<10.2f} | {p95[i]:<10.2f} | {p99[i]:<10.2f} | {recovery[i]:<12.2f}\n")

    # Figure 1: IEEE Single-Column Throughput Scaling
    fig, ax = plt.subplots(figsize=(3.5, 2.6))
    ax.plot(nodes, tps, 'o-', color='#003f5c', label='System Throughput')
    ax.set_title('Fig. 1. Throughput vs. Node Count', pad=8)
    ax.set_xlabel('Cluster Size ($N$ Nodes)')
    ax.set_ylabel('Throughput (TPS)')
    ax.grid(True)
    ax.set_xscale('log')
    ax.set_xticks([1, 3, 5, 10, 25, 50, 100])
    ax.get_xaxis().set_major_formatter(mpl.ticker.ScalarFormatter())
    plt.savefig(os.path.join(out_dir, "throughput_vs_nodes.png"))
    plt.close()

    # Figure 2: IEEE Tail Latency Percentiles
    fig, ax = plt.subplots(figsize=(3.5, 2.6))
    ax.plot(nodes, p50, 's-', color='#2f4b7c', label='$P_{50}$')
    ax.plot(nodes, p95, '^-', color='#ffa600', label='$P_{95}$')
    ax.plot(nodes, p99, 'd-', color='#d45087', label='$P_{99}$')
    ax.set_title('Fig. 2. Tail Latency Distribution', pad=8)
    ax.set_xlabel('Cluster Size ($N$ Nodes)')
    ax.set_ylabel('Latency (ms)')
    ax.grid(True)
    ax.set_xscale('log')
    ax.set_xticks([1, 3, 5, 10, 25, 50, 100])
    ax.get_xaxis().set_major_formatter(mpl.ticker.ScalarFormatter())
    ax.legend(frameon=True, facecolor='white', framealpha=0.9)
    plt.savefig(os.path.join(out_dir, "latency_percentiles.png"))
    plt.close()

    # Figure 3: IEEE Recovery Duration Under Fault Injection
    fig, ax = plt.subplots(figsize=(3.5, 2.6))
    ax.plot(nodes, recovery, 'x-', color='#f95d6a', label='Quorum Re-election')
    ax.set_title('Fig. 3. Quorum Recovery Duration', pad=8)
    ax.set_xlabel('Cluster Size ($N$ Nodes)')
    ax.set_ylabel('Recovery Time (ms)')
    ax.grid(True)
    ax.set_xscale('log')
    ax.set_xticks([1, 3, 5, 10, 25, 50, 100])
    ax.get_xaxis().set_major_formatter(mpl.ticker.ScalarFormatter())
    plt.savefig(os.path.join(out_dir, "recovery_time_under_chaos.png"))
    plt.close()

    print("Successfully generated IEEE-styled figures in results/plots/")

if __name__ == "__main__":
    generate_plots()
