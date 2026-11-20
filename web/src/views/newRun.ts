import { ApiError, startRun } from '../api';
import { h } from '../dom';
import type { RunRequest } from '../types';

export function renderNewRun(root: HTMLElement): () => void {
  const inputs: Record<string, HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement> = {};
  const errorEl = h('p', { class: 'error' });

  const labelled = (name: string, label: string, el: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement) => {
    inputs[name] = el;
    return h('label', {}, label, el);
  };
  const text = (name: string, label: string, value: string, type = 'text') =>
    labelled(name, label, h('input', { name, type, value }));
  const check = (name: string, label: string) => {
    const el = h('input', { name, type: 'checkbox' });
    inputs[name] = el;
    return h('label', { class: 'inline' }, el, ` ${label}`);
  };

  const method = h('select', { name: 'method' }, ...['GET', 'HEAD', 'POST', 'PUT', 'DELETE'].map((m) => h('option', { value: m }, m)));

  const str = (n: string) => (inputs[n] as HTMLInputElement).value;
  const num = (n: string) => Number((inputs[n] as HTMLInputElement).value);
  const chk = (n: string) => (inputs[n] as HTMLInputElement).checked;

  const form = h(
    'form',
    {
      onsubmit: async (e: Event) => {
        e.preventDefault();
        errorEl.textContent = '';
        Object.values(inputs).forEach((el) => el.classList.remove('invalid'));
        const req: RunRequest = {
          url: str('url'),
          method: str('method'),
          requests: num('requests'),
          rps: num('rps'),
          workers: num('workers'),
          timeout: num('timeout'),
          headers: str('headers'),
          body: str('body'),
          insecure: chk('insecure'),
          keep_alive: chk('keep_alive'),
          http2: chk('http2'),
          compression: chk('compression'),
          random_user_agent: chk('random_user_agent'),
        };
        try {
          const { id } = await startRun(req);
          location.hash = `#/runs/${id}`;
        } catch (err) {
          const e2 = err as ApiError;
          errorEl.textContent = e2.message;
          if (e2.field && inputs[e2.field]) {
            inputs[e2.field].classList.add('invalid');
            inputs[e2.field].focus();
          }
        }
      },
    },
    text('url', 'URL', '', 'url'),
    labelled('method', 'Method', method),
    h(
      'div',
      { class: 'row' },
      text('requests', 'Requests', '100', 'number'),
      text('rps', 'Requests / second', '10', 'number'),
      text('workers', 'Workers', '10', 'number'),
      text('timeout', 'Timeout (s)', '10', 'number'),
    ),
    text('headers', 'Headers (Key:Value,Key:Value)', ''),
    labelled('body', 'Body', h('textarea', { name: 'body', rows: 4 })),
    h(
      'div',
      { class: 'row' },
      check('insecure', 'Skip TLS verification'),
      check('keep_alive', 'Keep-alive'),
      check('http2', 'HTTP/2'),
      check('compression', 'Compression'),
      check('random_user_agent', 'Random User-Agent'),
    ),
    errorEl,
    h('button', { type: 'submit' }, 'Start run'),
  );

  root.append(h('h2', {}, 'New run'), form);
  return () => {};
}
