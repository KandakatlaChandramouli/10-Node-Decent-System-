#!/bin/bash
echo "version: '3'"
echo "services:"
for i in $(seq 0 99); do
    PEERS=""
    for j in $(seq 0 99); do
        if [ $i -ne $j ]; then
            PORT=$((8080 + j))
            [ -n "$PEERS" ] && PEERS="$PEERS,"
            PEERS="${PEERS}node${j}:${PORT}"
        fi
    done
    PORT=$((8080 + i))
    DPORT=$((9080 + i))
    cat <<SERVICE
  node${i}:
    build: .
    command: --port=${PORT} --peers=${PEERS}
    ports:
      - "${PORT}:${PORT}"
      - "${DPORT}:${DPORT}"
    networks:
      - sovnet
SERVICE
done
echo "networks:"
echo "  sovnet:"
echo "    driver: bridge"
