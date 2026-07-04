// Idiomatic JS wrapper around the morgan WebAssembly adapter.
//
//   import { loadMorgan } from './morgan.mjs';
//   const morgan = await loadMorgan();       // browser (fetch) or Node
//   morgan.format('tiny', {
//     method: 'GET', url: '/users/42', status: 200,
//     responseTime: 3.5, contentLength: 128,
//   });                                       // 'GET /users/42 200 128 - 3.500 ms'
//   morgan.formatDuration(1.5, 3);            // '1.500'
//
// The same Go implementation that powers the morgan module runs here via wasm.
// Only morgan's pure log-formatting helpers are exposed — the net/http
// middleware (morgan.New) is not portable to wasm and is not included.

async function ensureGo() {
  if (typeof globalThis.Go === 'function') return;
  if (typeof window === 'undefined') {
    // Node: wasm_exec.js is a classic script that assigns globalThis.Go.
    const { readFileSync } = await import('node:fs');
    const { fileURLToPath } = await import('node:url');
    const path = fileURLToPath(new URL('./wasm_exec.js', import.meta.url));
    const { runInThisContext } = await import('node:vm');
    runInThisContext(readFileSync(path, 'utf8'));
  } else {
    await import('./wasm_exec.js');
  }
}

async function readWasm(wasmPath) {
  if (typeof window === 'undefined') {
    const { readFileSync } = await import('node:fs');
    const { fileURLToPath } = await import('node:url');
    const p = wasmPath ?? fileURLToPath(new URL('./morgan.wasm', import.meta.url));
    return readFileSync(p);
  }
  const res = await fetch(wasmPath ?? new URL('./morgan.wasm', import.meta.url));
  return new Uint8Array(await res.arrayBuffer());
}

export async function loadMorgan(wasmPath) {
  await ensureGo();
  const go = new globalThis.Go();
  const bytes = await readWasm(wasmPath);
  const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
  go.run(instance); // long-running; resolves when the module exits (it won't)
  const g = globalThis.__mgo_morgan;
  if (!g) throw new Error('morgan wasm did not register __mgo_morgan');

  return {
    // format(nameOrFormatString, logObj) -> rendered log line.
    // nameOrFormatString may be a named format ('combined', 'common', 'short',
    // 'tiny') or a raw morgan format string using :token / :token[arg] syntax.
    // logObj fields (all optional): method, url, httpVersion, status, referrer,
    // remoteUser, userAgent, remoteAddr, pid, responseTime (ms), totalTime (ms),
    // incoming, date (epoch ms), contentLength.
    format: (nameOrFormat, log = {}) => g.format(String(nameOrFormat), log),
    // formatDuration(ms, digits=3) -> milliseconds string with fixed decimals.
    formatDuration: (ms, digits = 3) => g.formatDuration(Number(ms), Number(digits)),
    // tokens() -> array of built-in token names usable in format strings.
    tokens: () => g.tokens(),
    // formats() -> array of built-in named formats accepted by format().
    formats: () => g.formats(),
  };
}
