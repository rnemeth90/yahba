import { h } from '../dom';
import { formatBytes, formatDuration, formatMs } from '../format';
import type { Summary } from '../types';

const card = (label: string, value: string) =>
  h('div', { class: 'card' }, h('div', { class: 'card-label' }, label), h('div', { class: 'card-value' }, value));

export function summaryView(s?: Summary): HTMLElement {
  if (!s) return h('div');
  const codes = Object.entries(s.status_codes ?? {})
    .sort(([a], [b]) => Number(a) - Number(b))
    .map(([code, n]) => `${code === '0' ? 'conn. error' : code}: ${n}`)
    .join('   ');
  return h(
    'div',
    {},
    h(
      'div',
      { class: 'cards' },
      card('Requests', String(s.total_requests)),
      card('Failures', String(s.failures)),
      card('Req/s', s.rps.toFixed(1)),
      card('Avg', formatMs(s.avg_ms)),
      card('p50', formatMs(s.p50_ms)),
      card('p95', formatMs(s.p95_ms)),
      card('p99', formatMs(s.p99_ms)),
      card('Max', formatMs(s.max_ms)),
      card('Duration', formatDuration(s.duration_ms)),
      card('Received', formatBytes(s.bytes_received)),
    ),
    h('p', { class: 'codes' }, 'Status codes — ', codes || 'none'),
  );
}
