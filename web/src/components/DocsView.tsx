import { MORGAN } from '../data';

// DocsView links to the generated API reference, which is published alongside
// this landing site under /morgan/api/ (a relative ./api/ link).
export function DocsView() {
  const lib = MORGAN;
  return (
    <section className="view active" id="view-docs">
      <div className="sec-h"><span className="bar" /><h2 style={{ margin: 0 }}>API documentation</h2></div>
      <p className="muted">The full API reference for <code>{lib.pkg}</code> is generated from the source with a dependency-free <code>go/doc</code> tool (committed in the repo under <code>docs/gen</code>) and published alongside this site.</p>
      <div className="actions">
        <a className="pill b" href="./api/"><i className="fa-solid fa-book" /> Open the API reference</a>
        <a className="pill b" href={lib.repo} target="_blank" rel="noopener"><i className="fa-brands fa-github" />&nbsp;Source on GitHub</a>
      </div>
      <div className="note">The reference is browsable at <code>./api/</code> — every exported package, function, type and method, with source-linked signatures.</div>
    </section>
  );
}
