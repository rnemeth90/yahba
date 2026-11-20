import type { Run, RunDetail, RunRequest, Sample } from './types';

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public field?: string,
  ) {
    super(message);
  }
}

async function parse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string; field?: string };
    throw new ApiError(res.status, body.error ?? res.statusText, body.field);
  }
  return (await res.json()) as T;
}

export const startRun = (req: RunRequest) =>
  fetch('/api/runs', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  }).then((r) => parse<{ id: number }>(r));

export const listRuns = () => fetch('/api/runs').then((r) => parse<Run[]>(r));

export const getRun = (id: number) => fetch(`/api/runs/${id}`).then((r) => parse<RunDetail>(r));

export const stopRun = async (id: number): Promise<void> => {
  const res = await fetch(`/api/runs/${id}`, { method: 'DELETE' });
  if (!res.ok) await parse<unknown>(res);
};

export const getCompare = (a: number, b: number) =>
  fetch(`/api/compare?a=${a}&b=${b}`).then((r) => parse<{ a: RunDetail; b: RunDetail }>(r));

// streamRun replays past samples then follows live ones. EventSource reconnects
// on its own and the server replays, so callers must treat samples as idempotent by `t`.
export function streamRun(id: number, onSample: (s: Sample) => void, onDone: (run: Run) => void): () => void {
  const es = new EventSource(`/api/runs/${id}/events`);
  es.addEventListener('sample', (e) => onSample(JSON.parse((e as MessageEvent).data) as Sample));
  es.addEventListener('done', (e) => {
    es.close();
    onDone(JSON.parse((e as MessageEvent).data) as Run);
  });
  return () => es.close();
}
