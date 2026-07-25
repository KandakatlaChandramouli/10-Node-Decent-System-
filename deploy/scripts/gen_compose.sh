#!/bin/bash

NODES=${1:-10}
OUTPUT_FILE="deploy/docker/docker-compose-${NODES}.yml"

echo "version: '3.8'" > $OUTPUT_FILE
echo "services:" >> $OUTPUT_FILE

for i in $(seq 1 $NODES); do
  PORT=$((8080 + i - 1))
  PEERS=""
  for j in $(seq 1 $NODES); do
    if [ $i -ne $j ]; then
      PEER_PORT=$((8080 + j - 1))
      PEERS="${PEERS}- node-${j}:${PEER_PORT}\n      "
    fi
  done

  cat <<NODE_CONF >> $OUTPUT_FILE
  node-${i}:
    build:
      context: ../..
      dockerfile: deploy/docker/Dockerfile
    container_name: sovereign-node-${i}
    ports:
      - "${PORT}:${PORT}"
    environment:
      - NODE_ID=node-${i}
      - NODE_PORT=${PORT}
    networks:
      - sovereign-net
NODE_CONF
done

cat <<NET_CONF >> $OUTPUT_FILE

networks:
  sovereign-net:
    driver: bridge
NET_CONF

echo "Generated $OUTPUT_FILE with $NODES node configurations."
