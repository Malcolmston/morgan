// Library content for the morgan documentation site. This is the single
// `id:"morgan"` entry copied verbatim from the malcolmston/go landing site's
// data.ts, so the two stay in sync.
export interface Lib {
  id: string; name: string; icon: string; accent: string; pkg: string; node: string;
  repo: string; docs: string; tagline: string; blurb: string; tags: string[];
  features: string[]; node_code: string; go_code: string; integrate: string;
}

export const NODE_ACCENT = '#8cc84b';

export const MORGAN: Lib = {
  id:"morgan", name:"morgan", icon:'<i class="fa-solid fa-scroll"></i>', accent:"#f778ba",
  pkg:"github.com/malcolmston/morgan", node:"expressjs/morgan",
  repo:"https://github.com/malcolmston/morgan", docs:"https://malcolmston.github.io/morgan/",
  tagline:"HTTP request logger middleware.",
  blurb:"Wrap any http.Handler to log every request in a named format (combined, common, dev, short, tiny, json) "+
    "or your own :token string. Custom tokens, skip predicates, immediate mode and buffered output — just like the "+
    "Node original.",
  tags:["access logs","named formats","custom tokens","JSON","skip","buffering"],
  features:[
    "Named formats: <code>Combined</code>, <code>Common</code>, <code>Dev</code>, <code>Short</code>, <code>Tiny</code>, <code>JSON</code>",
    "Raw format strings with <code>:method :url :status :response-time</code> tokens",
    "Register custom tokens and named formats",
    "<code>Skip</code> predicate (e.g. only log errors), <code>Immediate</code> mode",
    "Buffered writes at an interval, any <code>io.Writer</code> stream",
    "Auto-detects a TTY to enable/disable dev colors"
  ],
  node_code:
`const morgan = require('morgan')
const express = require('express')
const app = express()

app.use(morgan('dev'))
app.listen(8080)`,
  go_code:
`import "github.com/malcolmston/morgan"

mux := http.NewServeMux()
mux.HandleFunc("/", handler)

http.ListenAndServe(":8080",
    morgan.New(mux, morgan.Dev, morgan.Config{}))`,
  integrate:
`<span class="tok-c">// Custom token, JSON output, only log errors</span>
morgan.Token("id", func(r *http.Request, log morgan.Log, args ...string) string {
    return r.Header.Get("X-Request-Id")
})

h := morgan.New(mux, morgan.JSON, morgan.Config{
    Skip: func(r *http.Request, status int) bool { return status < 400 },
})
http.ListenAndServe(":8080", h)`
};
