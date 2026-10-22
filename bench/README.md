# Benchmark comparison

`make bench` (or `bench/compare.sh`) runs yahba, wrk, ab and siege against the same local
target (`bench/target`, 1 KiB response) with identical workloads, then checks yahba against
tolerances. Tools that are not installed are skipped. Requires Go and python3.

- **Closed loop:** every tool uses `CONC` keep-alive connections for `DURATION` seconds. yahba
  runs in `--duration` mode. Results are the median of `RUNS` interleaved runs.
- **Open loop:** yahba is run at `PACED_RPS` and must achieve within 5% of it with no failures.
- **Pass criteria:** zero yahba failures, yahba throughput at least `100 - TOLERANCE` percent of
  the reference tool (default `wrk`), and the open-loop check above.

```
CONC=100 DURATION=15 RUNS=5 TOLERANCE=20 REFERENCE=ab make bench
```

Notes: siege is forced to keep-alive (it closes connections by default and exhausts ephemeral
ports on localhost); ab reports p99 in whole milliseconds; siege reports no percentiles. Run on
an otherwise idle machine, since client and server share the CPU.
