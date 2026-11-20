import { getRun, stopRun, streamRun } from '../api';
import { lineChart } from '../charts';
import { h } from '../dom';
import type { Run, Sample } from '../types';
import { summaryView } from './summary';

export function renderRun(root: HTMLElement, id: number): () => void {
  const samples = new Map<number, Sample>();
  const status = h('span', { class: 'badge' });
  const errorEl = h('p', { class: 'error' });
  const liveEl = h('p', { class: 'live' });
  const summaryEl = h('div');
  const rpsEl = h('div', { class: 'chart' });
  const latEl = h('div', { class: 'chart' });
  const stopBtn = h(
    'button',
    {
      type: 'button',
      class: 'danger',
      onclick: () => {
        stopRun(id).catch((e: Error) => {
          errorEl.textContent = e.message;
        });
      },
    },
    'Stop',
  );
  stopBtn.hidden = true;

  root.append(h('h2', {}, `Run #${id} `, status, ' ', stopBtn), errorEl, liveEl, summaryEl, rpsEl, latEl);

  const rps = lineChart(rpsEl, 'Requests per second', [{ label: 'req/s', stroke: '#2563eb' }], 'req/s');
  const lat = lineChart(
    latEl,
    'Latency',
    [
      { label: 'p50', stroke: '#16a34a' },
      { label: 'p95', stroke: '#d97706' },
      { label: 'p99', stroke: '#dc2626' },
    ],
    'ms',
  );

  const redraw = () => {
    const s = [...samples.values()].sort((a, b) => a.t - b.t);
    const xs = s.map((x) => x.t);
    rps.update(xs, [s.map((x) => x.count)]);
    lat.update(xs, [s.map((x) => x.p50_ms), s.map((x) => x.p95_ms), s.map((x) => x.p99_ms)]);
    const total = s.reduce((n, x) => n + x.count, 0);
    const failed = s.reduce((n, x) => n + x.errors, 0);
    liveEl.textContent = `${total} requests completed, ${failed} failed`;
  };

  const showRun = (run: Run) => {
    status.textContent = run.status;
    stopBtn.hidden = run.status !== 'running';
    summaryEl.replaceChildren(summaryView(run.summary));
  };

  let closeStream = () => {};
  let disposed = false;

  getRun(id)
    .then((d) => {
      if (disposed) return;
      d.samples.forEach((s) => samples.set(s.t, s));
      redraw();
      showRun(d.run);
      if (d.run.status === 'running') {
        closeStream = streamRun(
          id,
          (s) => {
            samples.set(s.t, s);
            redraw();
          },
          showRun,
        );
      }
    })
    .catch((e: Error) => {
      errorEl.textContent = e.message;
    });

  return () => {
    disposed = true;
    closeStream();
    rps.destroy();
    lat.destroy();
  };
}
