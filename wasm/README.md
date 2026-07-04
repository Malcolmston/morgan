# morgan — JavaScript adapter (WebAssembly)

Run the **same Go implementation** of morgan's log formatting from JavaScript —
in the browser or Node — via WebAssembly. No reimplementation: `main.go` exposes
morgan's portable, pure helpers to JS and `morgan.mjs` wraps them in an idiomatic
API.

Only the request-independent formatting is exposed. morgan's net/http middleware
(`morgan.New`, which wraps an `http.Handler`) is **not** portable to wasm and is
intentionally omitted — you bring your own log object.

## Build

```sh
./build.sh          # produces morgan.wasm (+ copies the Go wasm_exec.js runtime)
```

## Use (Node or browser)

```js
import { loadMorgan } from './morgan.mjs';
const morgan = await loadMorgan();

// Render a log line from a plain object using a named or raw format string.
morgan.format('tiny', {
  method: 'GET', url: '/users/42', status: 200,
  responseTime: 3.5, contentLength: 128,
});                                       // 'GET /users/42 200 128 - 3.500 ms'

morgan.format(':method :url -> :status', { method: 'POST', url: '/x', status: 201 });

morgan.formatDuration(1.5, 3);            // '1.500'
morgan.tokens();                          // ['http-version','method','url', ...]
morgan.formats();                         // ['combined','common','short','tiny']
```

`logObj` fields (all optional): `method`, `url`, `httpVersion`, `status`,
`referrer`, `remoteUser`, `userAgent`, `remoteAddr`, `pid`, `responseTime` (ms),
`totalTime` (ms), `incoming`, `date` (epoch ms), `contentLength`.

## Verify

```sh
./build.sh && node test.mjs
```

The adapter is compiled with `GOOS=js GOARCH=wasm`; on normal platforms `stub.go`
keeps `go build ./...` and CI green. Build artifacts (`*.wasm`) are gitignored.
