import re, matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt

with open("swarm.log") as f:
    log = f.read()

# Extract all latency values (ms)
latencies = [float(x) for x in re.findall(r'Latency: ([0-9.]+) ms', log)]

if latencies:
    plt.figure(figsize=(8,3))
    plt.plot(latencies, 'r-', alpha=0.7)
    plt.xlabel("Block sequence (accepted)")
    plt.ylabel("Latency (ms)")
    plt.title("Block Propagation Latency")
    plt.grid(True, alpha=0.3)
    plt.tight_layout()
    plt.savefig("propagation_latency.png", dpi=150)
    print(f"Saved latency graph – {len(latencies)} data points")
else:
    print("No latency data found in swarm.log")
