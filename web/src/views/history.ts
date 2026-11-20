import { listRuns } from '../api';
import { h } from '../dom';
import { formatMs } from '../format';

export function renderHistory(root: HTMLElement): () => void {
  const selected = new Set<number>();
  const errorEl = h('p', { class: 'error' });
  const body = h('tbody');
  const compareBtn = h(
    'button',
    {
      type: 'button',
      onclick: () => {
        const [a, b] = [...selected].sort((x, y) => x - y);
        location.hash = `#/compare?a=${a}&b=${b}`;
      },
    },
    'Compare selected',
  );
  compareBtn.disabled = true;

  root.append(
    h('h2', {}, 'History'),
    compareBtn,
    errorEl,
    h(
      'table',
      {},
      h('thead', {}, h('tr', {}, ...['', 'Run', 'Started', 'Status', 'URL', 'Requests', 'p95', 'Req/s'].map((c) => h('th', {}, c)))),
      body,
    ),
  );

  listRuns()
    .then((runs) => {
      if (runs.length === 0) body.append(h('tr', {}, h('td', { colspan: 8 }, 'No runs yet.')));
      for (const run of runs) {
        const box = h('input', {
          type: 'checkbox',
          onchange: (e: Event) => {
            if ((e.target as HTMLInputElement).checked) selected.add(run.id);
            else selected.delete(run.id);
            compareBtn.disabled = selected.size !== 2;
          },
        });
        const s = run.summary;
        body.append(
          h(
            'tr',
            {},
            h('td', {}, box),
            h('td', {}, h('a', { href: `#/runs/${run.id}` }, `#${run.id}`)),
            h('td', {}, new Date(run.started_at).toLocaleString()),
            h('td', {}, run.status),
            h('td', {}, run.config.url),
            h('td', {}, s ? String(s.total_requests) : '–'),
            h('td', {}, s ? formatMs(s.p95_ms) : '–'),
            h('td', {}, s ? s.rps.toFixed(1) : '–'),
          ),
        );
      }
    })
    .catch((e: Error) => {
      errorEl.textContent = e.message;
    });

  return () => {};
}
