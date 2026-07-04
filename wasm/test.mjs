// Node smoke test: builds must be run first (see build.sh). Verifies the Go
// implementation is reachable from JS through wasm.
import assert from 'node:assert';
import { loadMorgan } from './morgan.mjs';

const morgan = await loadMorgan();

const line = morgan.format('tiny', {
  method: 'GET',
  url: '/users/42',
  status: 200,
  responseTime: 3.5,
  contentLength: 128,
});
assert.ok(line.includes('GET'), 'tiny output should contain the method');
assert.ok(line.includes('/users/42'), 'tiny output should contain the url');
assert.ok(line.includes('200'), 'tiny output should contain the status');

assert.strictEqual(morgan.formatDuration(1.5, 3), '1.500', 'formatDuration(1.5, 3)');

assert.ok(
  Array.isArray(morgan.tokens()) && morgan.tokens().includes('method'),
  'tokens() should list built-in token names',
);
assert.ok(
  Array.isArray(morgan.formats()) && morgan.formats().includes('tiny'),
  'formats() should list named formats',
);

console.log('morgan wasm adapter: all JS-side assertions passed');
console.log(line);
process.exit(0);
