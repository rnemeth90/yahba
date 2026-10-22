#!/usr/bin/env python3
"""Parses raw tool output written by compare.sh, prints a comparison table and
exits non-zero if yahba falls outside the tolerances."""
import argparse
import glob
import json
import os
import re
import statistics
import sys

UNITS_MS = {"ns": 1e-6, "us": 1e-3, "µs": 1e-3, "ms": 1.0, "s": 1000.0, "m": 60000.0}


def to_ms(text):
    """Converts '911.917µs' or '1.5ms' (Go and wrk formats) to milliseconds."""
    m = re.fullmatch(r"([\d.]+)\s*(ns|us|µs|ms|s|m)", text.strip())
    return float(m.group(1)) * UNITS_MS[m.group(2)] if m else None


def parse_yahba(text):
    d = json.loads(text)
    return {
        "rps": d["achieved_rps"],
        "requests": d["total_requests"],
        "failures": d["failures"],
        "p99": to_ms(d["latency"]["p99"]),
    }


def parse_wrk(text):
    rps = float(re.search(r"Requests/sec:\s+([\d.]+)", text).group(1))
    requests = int(re.search(r"(\d+) requests in", text).group(1))
    p99 = re.search(r"99%\s+([\d.]+\w+)", text)
    non2xx = re.search(r"Non-2xx or 3xx responses:\s+(\d+)", text)
    errs = re.search(r"Socket errors: connect (\d+), read (\d+), write (\d+), timeout (\d+)", text)
    failures = int(non2xx.group(1)) if non2xx else 0
    if errs:
        failures += sum(int(x) for x in errs.groups())
    return {"rps": rps, "requests": requests, "failures": failures, "p99": to_ms(p99.group(1)) if p99 else None}


def parse_ab(text):
    rps = float(re.search(r"Requests per second:\s+([\d.]+)", text).group(1))
    requests = int(re.search(r"Complete requests:\s+(\d+)", text).group(1))
    failed = int(re.search(r"Failed requests:\s+(\d+)", text).group(1))
    non2xx = re.search(r"Non-2xx responses:\s+(\d+)", text)
    p99 = re.search(r"99%\s+(\d+)", text)
    return {
        "rps": rps,
        "requests": requests,
        "failures": failed + (int(non2xx.group(1)) if non2xx else 0),
        "p99": float(p99.group(1)) if p99 else None,  # ab reports whole milliseconds
    }


def parse_siege(text):
    rps = float(re.search(r"Transaction rate:\s+([\d.]+)", text).group(1))
    ok = int(re.search(r"Successful transactions:\s+(\d+)", text).group(1))
    failed = int(re.search(r"Failed transactions:\s+(\d+)", text).group(1))
    return {"rps": rps, "requests": ok + failed, "failures": failed, "p99": None}


PARSERS = {"yahba": parse_yahba, "wrk": parse_wrk, "ab": parse_ab, "siege": parse_siege}


def load(out_dir):
    runs = {}
    for tool, parser in PARSERS.items():
        for path in sorted(glob.glob(os.path.join(out_dir, f"{tool}-*.txt"))):
            try:
                runs.setdefault(tool, []).append(parser(open(path).read()))
            except (AttributeError, ValueError, KeyError):
                print(f"warning: could not parse {os.path.basename(path)}", file=sys.stderr)
    return runs


def median(rows, key):
    vals = [r[key] for r in rows if r[key] is not None]
    return statistics.median(vals) if vals else None


def fmt(v, spec):
    return "-" if v is None else format(v, spec)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("out_dir")
    ap.add_argument("--tolerance", type=float, default=15, help="max percent yahba may trail the reference")
    ap.add_argument("--reference")
    ap.add_argument("--paced-rps", type=int, default=0)
    args = ap.parse_args()

    runs = load(args.out_dir)
    if "yahba" not in runs:
        print("FAIL: no parsable yahba results")
        return 1

    meds = {t: {k: median(rows, k) for k in ("rps", "requests", "failures", "p99")} for t, rows in runs.items()}
    others = [t for t in runs if t != "yahba"]
    ref = args.reference if args.reference in runs else (others[0] if others else None)

    print(f"\n{'tool':<8} {'req/s (median)':>16} {'vs ref':>8} {'requests':>10} {'failures':>9} {'p99 ms':>9}")
    for tool in ["yahba"] + others:
        m = meds[tool]
        ratio = m["rps"] / meds[ref]["rps"] * 100 if ref else None
        print(f"{tool:<8} {m['rps']:>16.0f} {fmt(ratio, '.0f') + '%':>8} {m['requests']:>10.0f} {m['failures']:>9.0f} {fmt(m['p99'], '.3f'):>9}")
    print(f"(reference: {ref or 'none'}; ab p99 has 1 ms resolution; siege reports no percentiles)\n")

    failed = []
    if meds["yahba"]["failures"] > 0:
        failed.append(f"yahba reported {meds['yahba']['failures']:.0f} failures against a healthy target")
    if ref:
        floor = 100 - args.tolerance
        pct = meds["yahba"]["rps"] / meds[ref]["rps"] * 100
        if pct < floor:
            failed.append(f"yahba throughput is {pct:.0f}% of {ref}, below the {floor:.0f}% floor")

    paced_path = os.path.join(args.out_dir, "paced-1.txt")
    if args.paced_rps and os.path.exists(paced_path):
        try:
            d = json.load(open(paced_path))
            got, want = d["achieved_rps"], args.paced_rps
            print(f"open loop: configured {want} RPS, achieved {got:.1f} RPS, failures {d['failures']}")
            if abs(got - want) / want > 0.05:
                failed.append(f"paced mode achieved {got:.1f} RPS, more than 5% off the configured {want}")
            if d["failures"] > 0:
                failed.append(f"paced mode reported {d['failures']} failures")
        except (ValueError, KeyError):
            failed.append("could not parse paced-mode output")

    if failed:
        print("\nFAIL")
        for f in failed:
            print(f"  - {f}")
        return 1
    print("\nPASS")
    return 0


if __name__ == "__main__":
    sys.exit(main())
