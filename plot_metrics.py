import json, time, matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt

with open("chain.json") as f:
    blocks = json.load(f)

# Parse timestamps
times = []
for b in blocks:
    try:
        t = time.strptime(b["timestamp"][:26], "%Y-%m-%dT%H:%M:%S.%f")
        times.append(time.mktime(t))
    except:
        pass

# Block interval
if len(times) > 1:
    intervals = [times[i]-times[i-1] for i in range(1, len(times))]
    plt.figure(figsize=(8,3))
    plt.plot(intervals)
    plt.xlabel("Block sequence")
    plt.ylabel("Interval (s)")
    plt.title("Block Interval")
    plt.grid(True, alpha=0.3)
    plt.tight_layout()
    plt.savefig("block_interval.png", dpi=150)
    plt.close()

# Transactions per block
tx_counts = [len(b["transactions"]) for b in blocks]
plt.figure(figsize=(8,3))
plt.bar(range(len(tx_counts)), tx_counts, color='purple', alpha=0.7)
plt.xlabel("Block index")
plt.ylabel("Transactions")
plt.title("Transactions per Block")
plt.grid(True, alpha=0.3, axis='y')
plt.tight_layout()
plt.savefig("tx_per_block.png", dpi=150)
plt.close()

print("Graphs saved: block_interval.png, tx_per_block.png")
