#!/usr/bin/env bash
# run-keepalive-test.sh — starts nginx and streams its access log.
# Press Ctrl+C to stop.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== Starting nginx ==="
docker compose -f "$SCRIPT_DIR/docker-compose.yml" up -d --wait 2>&1
CONTAINER=$(docker compose -f "$SCRIPT_DIR/docker-compose.yml" ps -q nginx)

for i in $(seq 1 10); do
    if curl -sf http://localhost:8088/ >/dev/null 2>&1; then
        echo "    nginx is up at http://localhost:8088/"
        break
    fi
    echo "    attempt $i — waiting..."
    sleep 1
done

cleanup() {
    echo ""
    echo "=== Stopping nginx ==="
    docker compose -f "$SCRIPT_DIR/docker-compose.yml" down
}
trap cleanup EXIT

echo ""
echo "=== Streaming nginx access log (Ctrl+C to stop) ==="
echo "    conn=<serial>  reqs=<requests-on-this-connection>"
echo ""
docker logs -f "$CONTAINER" 2>&1

# reload nginx once:
#   docker exec $(docker compose -f test/docker-compose.yml ps -q nginx) nginx -s reload
#
# reload nginx 100 times:
#   for i in {1..1000}; do docker exec $(docker compose -f test/docker-compose.yml ps -q nginx) nginx -s reload; done
#   for i in $(seq 1 1000); do nginx -s reload; sleep 1; done             

