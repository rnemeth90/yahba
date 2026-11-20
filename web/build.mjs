import { build } from 'esbuild';

await build({
  entryPoints: ['src/main.ts'],
  bundle: true,
  minify: true,
  target: 'es2020',
  outfile: '../internal/web/static/app.js', // CSS imported from TS lands next to it as app.css
  loader: { '.css': 'css' },
  logLevel: 'info',
});
