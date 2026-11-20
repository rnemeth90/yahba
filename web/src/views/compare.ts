import { getCompare } from '../api';
import { lineChart } from '../charts';
import { alignSeries, compareSummaries, type Delta } from '../compare';
import { h } from '../dom';
import { formatMs } from '../format';

const fmt = (v: number, unit: Delta['unit']) =>
  unit === 'ms' ? formatMs(v) : unit === 'rps' ? v.toFixed(1) : String(v);

export function renderCompare(root: HTMLElement, a: number, b: number): () => void {
  const errorEl = h('p', { class: 'error' });
  const tableEl = h('div');
  const rpsEl = h('div', { class: 'chart' });
  const latEl = h('div', { class: 'chart' });
  root.append(h('h2', {}, `Run #${a} vs #${b}`), errorEl, tableEl, rpsEl, latEl);

  const rps = lineChart(rpsEl, 'Requests per second', [
    { label: `#${a}`, stroke: '#2563eb' },
    { label: `#${b}`, stroke: '#dc2626', dash: [6, 4] },
  ], 'req/s');
  const lat = lineChart(latEl, 'p95 latency', [
    { label: `#${a}`, stroke: '#2563eb' },
    { label: `#${b}`, stroke: '#dc2626', dash: [6, 4] },
  ], 'ms');

  let disposed = false;
  getCompare(a, b)
    .then(({ a: da, b: db }) => {
      if (disposed) return;
      const r = alignSeries(da.samples, db.samples, (s) => s.count);
      rps.update(r.xs, [r.a, r.b]);
      const l = alignSeries(da.samples, db.samples, (s) => s.p95_ms);
      lat.update(l.xs, [l.a, l.b]);

      if (!da.run.summary || !db.run.summary) {
        tableEl.textContent = 'Both runs must be finished to compare totals.';
        return;
      }
      const rows = compareSummaries(da.run.summary, db.run.summary).map((d) => {
        const better = d.diff === 0 ? '' : (d.diff < 0) === d.lowerIsBetter ? 'better' : 'worse';
        const pct = d.pct === null ? '–' : `${d.pct > 0 ? '+' : ''}${d.pct.toFixed(1)}%`;
        return h(
          'tr',
          {},
          h('td', {}, d.label),
          h('td', {}, fmt(d.a, d.unit)),
          h('td', {}, fmt(d.b, d.unit)),
          h('td', { class: better }, pct),
        );
      });
      tableEl.replaceChildren(
        h(
          'table',
          {},
          h('thead', {}, h('tr', {}, ...['', `#${a}`, `#${b}`, 'Change'].map((c) => h('th', {}, c)))),
          h('tbody', {}, ...rows),
        ),
      );
    })
    .catch((e: Error) => {
      errorEl.textContent = e.message;
    });

  return () => {
    disposed = true;
    rps.destroy();
    lat.destroy();
  };
}
