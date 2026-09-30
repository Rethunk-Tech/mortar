const isEl = (c: Element): c is HTMLElement => 'inert' in c

// While the overlay is up, what it covers is inert so focus cannot reach it; a stable function so the ref runs once.
export function holdFocus(el: HTMLElement) {
  const covered: HTMLElement[] = []
  const mark = (c: HTMLElement) => {
    if (c.inert) {
      return
    }
    const keep =
      c.dataset.windowControls === undefined ? c.querySelector('[data-window-controls]') : c
    if (keep && c.contains(keep)) {
      for (const child of c.children) {
        if (child !== keep && isEl(child) && !keep.contains(child)) {
          mark(child)
        }
      }
      return
    }
    c.inert = true
    covered.push(c)
  }
  let node: HTMLElement | null = el
  while (node) {
    const parent: HTMLElement | null = node.parentElement
    if (!parent) {
      break
    }
    for (const c of parent.children) {
      if (c !== node && isEl(c)) {
        mark(c)
      }
    }
    if ((typeof document !== 'undefined' && parent === document.body) || parent.id === 'root') {
      break
    }
    node = parent
  }
  return () => {
    for (const c of covered) {
      c.inert = false
    }
  }
}
