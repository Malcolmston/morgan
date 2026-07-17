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
  tagline:"HTTP request logger middleware for net/http.",
  blurb:"Wrap any http.Handler to log every request in one of six named formats (combined, common, dev, short, tiny, json) "+
    "or your own :token string. Fifteen built-in tokens, custom tokens and named formats, skip predicates, immediate "+
    "mode and buffered output — behavioural parity with the Node original, depending only on the standard library.",
  tags:["access logs","net/http","named formats","custom tokens","JSON","skip","buffering","stdlib-only"],
  features:[
    "Six named formats: <code>Combined</code>, <code>Common</code>, <code>Dev</code>, <code>Short</code>, <code>Tiny</code>, <code>JSON</code>",
    "15 built-in tokens: <code>:method</code> <code>:url</code> <code>:status</code> <code>:response-time</code> <code>:total-time</code> <code>:date[clf]</code> <code>:req[header]</code> <code>:res[header]</code> and more",
    "<code>Compile</code> a raw format string into a reusable <code>FormatFunc</code>",
    "Register custom tokens with <code>Token</code> / <code>TokenFunc</code>",
    "Register named formats with <code>RegisterFormat</code> and <code>RegisterFormatFunc</code>",
    "<code>Config.Skip</code> predicate (e.g. only log errors) and <code>Config.Immediate</code> mode",
    "Buffered writes at an interval (<code>Config.Buffer</code>) to any <code>io.Writer</code> (<code>Config.Stream</code>)",
    "<code>Dev</code> colors by status class and auto-detects a TTY, falling back to plain output when piped",
    "Assemble a <code>Log</code> yourself with <code>FromRequest</code> and <code>FormatDuration</code>",
    "Zero dependencies — standard library only"
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
