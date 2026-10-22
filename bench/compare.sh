#!/usr/bin/env bash
# Compares yahba against other load testing tools (wrk, ab, siege) on the same
# target and workload, then checks yahba against tolerances.
#
# Every tool runs closed loop with HTTP keep-alive: CONC concurrent connections
# for DURATION seconds. yahba additionally gets an open-loop accuracy check.
#
# wrk is a multithreaded C client and sets the ceiling, so the default tolerance
# (yahba may trail the reference by TOLERANCE percent) is generous.
#
# Usage: bench/compare.sh
# Env:   CONC=50 DURATION=10 RUNS=3 TOLERANCE=25 PACED_RPS=2000 PORT=8098
#        REFERENCE=wrk   (tool yahba is compared to; default: first one found)
#        TOOLS="wrk ab siege"  (subset to run; missing tools are skipped)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONC="${CONC:-50}"
DURATION="${DURATION:-10}"
RUNS="${RUNS:-3}"
TOLERANCE="${TOLERANCE:-25}"
PACED_RPS="${PACED_RPS:-2000}"
PORT="${PORT:-8098}"
TOOLS="${TOOLS:-wrk ab siege}"
URL="http://127.0.0.1:${PORT}/"

OUT="$(mktemp -d)"
TARGET_PID=""
cleanup() {
    if [ -n "$TARGET_PID" ]; then
        kill "$TARGET_PID" 2>/dev/null || true
        wait "$TARGET_PID" 2>/dev/null || true
    fi
    rm -rf "$OUT"
}
trap cleanup EXIT

ulimit -n 10240 2>/dev/null || true

echo "=== Building yahba and target ==="
(cd "$ROOT" && go build -o "$OUT/yahba" . && go build -o "$OUT/target" ./bench/target)

"$OUT/target" -addr "127.0.0.1:${PORT}" &
TARGET_PID=$!
for _ in $(seq 1 20); do
    curl -sf "$URL" >/dev/null 2>&1 && break
    sleep 0.25
done
curl -sf "$URL" >/dev/null || { echo "target did not start"; exit 1; }

# siege closes connections by default, which exhausts ephemeral ports
printf 'connection = keep-alive\nverbose = false\nlogging = false\nshow-logfile = false\n' >"$OUT/siegerc"

available=()
for t in yahba $TOOLS; do
    if [ "$t" = yahba ] || command -v "$t" >/dev/null 2>&1; then
        available+=("$t")
    else
        echo "skipping $t (not installed)"
    fi
done

run_tool() {
    case "$1" in
    yahba) "$OUT/yahba" run -u "$URL" --duration "${DURATION}s" -w "$CONC" -k -f json ;;
    wrk)
        threads=$((CONC < 4 ? CONC : 4))
        wrk -t"$threads" -c "$CONC" -d "${DURATION}s" --latency "$URL"
        ;;
    ab) ab -k -c "$CONC" -t "$DURATION" -n 100000000 "$URL" ;;
    siege) SIEGERC="$OUT/siegerc" siege -b -c "$CONC" -t "${DURATION}S" "$URL" 2>&1 ;;
    esac
}

echo "=== Warmup ==="
"$OUT/yahba" run -u "$URL" --duration 2s -w "$CONC" -k -f json >/dev/null

echo "=== Closed loop: ${CONC} connections, ${DURATION}s, ${RUNS} runs per tool ==="
for i in $(seq 1 "$RUNS"); do
    for t in "${available[@]}"; do
        echo "  run $i: $t"
        run_tool "$t" >"$OUT/$t-$i.txt" 2>&1 || true
        sleep 1
    done
done

echo "=== Open loop: yahba at ${PACED_RPS} RPS ==="
"$OUT/yahba" run -u "$URL" --rps "$PACED_RPS" -r $((PACED_RPS * DURATION)) -w "$CONC" -k -f json >"$OUT/paced-1.txt" 2>&1 || true

python3 "$ROOT/bench/report.py" "$OUT" --tolerance "$TOLERANCE" --paced-rps "$PACED_RPS" ${REFERENCE:+--reference "$REFERENCE"}
