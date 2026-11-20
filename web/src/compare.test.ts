import { describe, expect, it } from 'vitest';
import { alignSeries, compareSummaries } from './compare';
import type { Sample, Summary } from './types';

const base: Summary = {
  host: 'h', method: 'GET', total_requests: 100, successes: 100, failures: 0,
  duration_ms: 10000, rps: 10, min_ms: 1, avg_ms: 10, p50_ms: 8, p95_ms: 20,
  p99_ms: 30, max_ms: 40, status_codes: {}, bytes_sent: 0, bytes_received: 0,
};

const smp = (t: number, p95: number): Sample => ({
  t, count: 1, errors: 0, avg_ms: 0, p50_ms: 0, p95_ms: p95, p99_ms: 0, status_codes: {},
});

describe('compareSummaries', () => {
  it('computes absolute and percent deltas', () => {
    const rows = compareSummaries(base, { ...base, p95_ms: 30, rps: 5 });
    const p95 = rows.find((r) => r.label === 'p95')!;
    expect(p95).toMatchObject({ a: 20, b: 30, diff: 10, pct: 50, lowerIsBetter: true, unit: 'ms' });
    const rps = rows.find((r) => r.label === 'Req/s')!;
    expect(rps).toMatchObject({ diff: -5, pct: -50, lowerIsBetter: false, unit: 'rps' });
  });

  it('returns null pct when the baseline is zero', () => {
    const rows = compareSummaries({ ...base, failures: 0 }, { ...base, failures: 3 });
    expect(rows.find((r) => r.label === 'Failures')!.pct).toBeNull();
  });
});

describe('alignSeries', () => {
  it('unions x values and fills gaps with null', () => {
    const out = alignSeries([smp(0, 1), smp(1, 2)], [smp(1, 5), smp(2, 6)], (s) => s.p95_ms);
    expect(out.xs).toEqual([0, 1, 2]);
    expect(out.a).toEqual([1, 2, null]);
    expect(out.b).toEqual([null, 5, 6]);
  });
});
