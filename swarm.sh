#!/bin/bash
cd /workspaces/10-Node-Decent-System-

BASE_PORT=8080
NODES=10

PEERS=""
for (( i=0; i<NODES; i++ )); do
    PORT=$((BASE_PORT + i))
    if [ -z "$PEERS" ]; then
        PEERS="127.0.0.1:$PORT"
    else
        PEERS="$PEERS,127.0.0.1:$PORT"
    fi
done

touch swarm.log
echo "Starting $NODES nodes with peers: $PEERS"

for (( i=0; i<NODES; i++ )); do
    PORT=$((BASE_PORT + i))
    stdbuf -oL go run main.go --port=$PORT --peers="$PEERS" >> swarm.log 2>&1 &
    echo "Node on port $PORT (PID $!)"
done

echo "All nodes launched. Watching logs..."
tail -f swarm.log
