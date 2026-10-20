// Installed before the app loads: React reports every commit to a DevTools hook, production builds included. A
// component counts when React did work on it in that commit (PerformedWork), as a mount without a previous fiber or as
// an update with one; subtrees React reused untouched are not walked.
export const HOOK = `(() => {
  const st = { commits: 0, mounts: 0, updates: 0, names: {}, roots: {}, why: {}, mutations: 0, longTasks: 0, longMs: 0, t0: performance.now(), last: 0 }
  const COMPONENT = new Set([0, 1, 11, 14, 15])
  const nameOf = (f) => {
    const t = f.type
    if (typeof t === 'function') return t.displayName || t.name || 'anonymous'
    const inner = t && (t.type || t.render)
    return (inner && (inner.displayName || inner.name)) || (t && t.displayName) || 'anonymous'
  }
  // Why a cascade started: its root's props that changed, or the state hooks whose value changed, by position.
  const peek = (v) => {
    if (v === null || typeof v !== 'object') return typeof v === 'function' ? 'fn' : String(v).slice(0, 24)
    if (Array.isArray(v)) return 'array(' + v.length + ')'
    return '{' + Object.keys(v).slice(0, 3).join(',') + '}'
  }
  const noteWhy = (f, a, n) => {
    const out = []
    if (f.memoizedProps !== a.memoizedProps && f.memoizedProps && a.memoizedProps) {
      for (const k of Object.keys(f.memoizedProps)) if (f.memoizedProps[k] !== a.memoizedProps[k]) out.push('prop ' + k)
    }
    for (let x = f.memoizedState, y = a.memoizedState, i = 0; x && y && out.length < 3; x = x.next, y = y.next, i++) {
      if (x.queue && x.memoizedState !== y.memoizedState) out.push('hook ' + i + ' ' + peek(x.memoizedState))
    }
    return n + ': ' + (out.join('; ') || 'context or parent')
  }
  // root: why the cascade of updates started, at the component it started at (one whose nearest component above did
  // not render), which every update in the cascade is counted against; null while walking components React did not
  // touch.
  const walk = (f, root) => {
    for (; f; f = f.sibling) {
      const a = f.alternate
      const component = COMPONENT.has(f.tag)
      const worked = component ? (f.flags & 1) !== 0 : root !== null
      let here = root
      if (component) {
        here = null
        if (worked && a) {
          const n = nameOf(f)
          here = root ?? noteWhy(f, a, n)
          const started = here.slice(0, here.indexOf(': '))
          st.updates++
          st.names[n] = (st.names[n] || 0) + 1
          st.roots[started] = (st.roots[started] || 0) + 1
          st.why[here] = (st.why[here] || 0) + 1
        } else if (worked) {
          st.mounts++
          here = root
        }
      }
      if (f.child && (!a || a.child !== f.child)) walk(f.child, here)
    }
  }
  window.__REACT_DEVTOOLS_GLOBAL_HOOK__ = {
    supportsFiber: true,
    renderers: new Map(),
    inject(r) { this.renderers.set(this.renderers.size + 1, r); return this.renderers.size },
    onScheduleFiberRoot() {},
    onCommitFiberRoot(_id, root) { st.commits++; st.last = performance.now(); try { walk(root.current.child, null) } catch {} },
    onCommitFiberUnmount() {},
    onPostCommitFiberRoot() {},
    checkDCE() {},
  }
  new PerformanceObserver((list) => {
    for (const e of list.getEntries()) { st.longTasks++; st.longMs += e.duration }
  }).observe({ type: 'longtask', buffered: true })
  document.addEventListener('DOMContentLoaded', () => {
    new MutationObserver((m) => { st.mutations += m.length }).observe(document.documentElement, {
      subtree: true, childList: true, attributes: true, characterData: true,
    })
  })
  window.mortarRenders = {
    reset() { Object.assign(st, { commits: 0, mounts: 0, updates: 0, names: {}, roots: {}, why: {}, mutations: 0, longTasks: 0, longMs: 0, t0: performance.now(), last: 0 }) },
    // settledMs: from the reset to the last commit, the wall time until the screen stopped changing.
    read() { return { ...st, settledMs: st.last > 0 ? st.last - st.t0 : 0, names: { ...st.names }, roots: { ...st.roots }, why: { ...st.why } } },
  }
})()`
