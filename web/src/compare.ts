import type { Sample, Summary } from './types';

export interface Delta {
  label: string;
  a: number;
  b: number;
  diff: number;
  pct: number | null;
  lowerIsBetter: boolean;
  unit: 'ms' | 'rps' | 'count';
}

const rows: { label: string; pick: (s: Summary) => number; lowerIsBetter: boolean; unit: Delta['unit'] }[] = [
  { label: 'Req/s', pick: (s) => s.rps, lowerIsBetter: false, unit: 'rps' },
  { label: 'Avg', pick: (s) => s.avg_ms, lowerIsBetter: true, unit: 'ms' },
  { label: 'p50', pick: (s) => s.p50_ms, lowerIsBetter: true, unit: 'ms' },
  { label: 'p95', pick: (s) => s.p95_ms, lowerIsBetter: true, unit: 'ms' },
  { label: 'p99', pick: (s) => s.p99_ms, lowerIsBetter: true, unit: 'ms' },
  { label: 'Max', pick: (s) => s.max_ms, lowerIsBetter: true, unit: 'ms' },
  { label: 'Failures', pick: (s) => s.failures, lowerIsBetter: true, unit: 'count' },
];

export function compareSummaries(a: Summary, b: Summary): Delta[] {
  return rows.map(({ label, pick, lowerIsBetter, unit }) => {
    const av = pick(a);
    const bv = pick(b);
    const diff = bv - av;
    return { label, a: av, b: bv, diff, pct: av === 0 ? null : (diff / av) * 100, lowerIsBetter, unit };
  });
}

export function alignSeries(
  a: Sample[],
  b: Sample[],
  pick: (s: Sample) => number,
): { xs: number[]; a: (number | null)[]; b: (number | null)[] } {
  const xs = [...new Set([...a, ...b].map((s) => s.t))].sort((x, y) => x - y);
  const ma = new Map(a.map((s) => [s.t, pick(s)]));
  const mb = new Map(b.map((s) => [s.t, pick(s)]));
  return { xs, a: xs.map((x) => ma.get(x) ?? null), b: xs.map((x) => mb.get(x) ?? null) };
}
