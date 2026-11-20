import './style.css';
import { renderCompare } from './views/compare';
import { renderHistory } from './views/history';
import { renderNewRun } from './views/newRun';
import { renderRun } from './views/run';

const app = document.getElementById('app') as HTMLElement;
let cleanup: () => void = () => {};

function route(): void {
  cleanup();
  app.replaceChildren();
  const [path, query = ''] = location.hash.slice(1).split('?');
  const params = new URLSearchParams(query);
  const run = path.match(/^\/runs\/(\d+)$/);
  if (run) cleanup = renderRun(app, Number(run[1]));
  else if (path === '/history') cleanup = renderHistory(app);
  else if (path === '/compare') cleanup = renderCompare(app, Number(params.get('a')), Number(params.get('b')));
  else cleanup = renderNewRun(app);
}

window.addEventListener('hashchange', route);
route();
