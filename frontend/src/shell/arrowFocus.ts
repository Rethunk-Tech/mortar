import type { KeyboardEvent } from 'react'

interface Rect {
  top: number
  bottom: number
  left: number
  right: number
}

const ROW_SLACK_PX = 4

// nextIndex is the item an arrow key moves to from item at: Left and Right step through the items in order, Up and
// Down go to the nearest row above or below and, in it, the item nearest by horizontal centre. -1 means stay.
function nextIndex(rects: readonly Rect[], at: number, key: string): number {
  const cur = rects[at]
  if (!cur) {
    return -1
  }
  if (key === 'ArrowLeft' || key === 'ArrowRight') {
    const next = at + (key === 'ArrowLeft' ? -1 : 1)
    return next >= 0 && next < rects.length ? next : -1
  }
  const down = key === 'ArrowDown'
  if (!down && key !== 'ArrowUp') {
    return -1
  }
  const centre = (r: Rect) => (r.left + r.right) / 2
  let best = -1
  let bestRow = Number.POSITIVE_INFINITY
  let bestX = Number.POSITIVE_INFINITY
  rects.forEach((r, i) => {
    const gap = down ? r.top - cur.bottom : cur.top - r.bottom
    if (i === at || gap < -ROW_SLACK_PX) {
      return
    }
    const row = Math.abs(r.top - cur.top)
    const x = Math.abs(centre(r) - centre(cur))
    if (row < bestRow - ROW_SLACK_PX || (Math.abs(row - bestRow) <= ROW_SLACK_PX && x < bestX)) {
      best = i
      bestRow = row
      bestX = x
    }
  })
  return best
}

const FOCUSABLE =
  'button:not(:disabled), a[href], [tabindex]:not([tabindex="-1"]), input, select, textarea'

function editing(el: EventTarget | null): boolean {
  return (
    el instanceof HTMLElement &&
    (el.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName))
  )
}

// arrowFocus walks a grid or row of items with the arrow keys, which a controller sends under Steam Input. items
// selects the items inside the handler's element; pick chooses what to focus in the item reached (by default the
// item itself when focusable, else its first control). Text fields and handlers that already took the key keep it.
function arrowFocus(
  e: KeyboardEvent<HTMLElement>,
  items: string,
  pick: (item: HTMLElement) => HTMLElement | null = (item) =>
    item.matches(FOCUSABLE) ? item : item.querySelector<HTMLElement>(FOCUSABLE),
): void {
  if (e.defaultPrevented || editing(e.target)) {
    return
  }
  const list = [...e.currentTarget.querySelectorAll<HTMLElement>(items)]
  const at = list.findIndex((item) => item.contains(document.activeElement))
  const next = nextIndex(
    list.map((item) => item.getBoundingClientRect()),
    at,
    e.key,
  )
  const target = next < 0 ? null : pick(list[next] as HTMLElement)
  if (target) {
    e.preventDefault()
    target.focus()
  }
}

export { arrowFocus, nextIndex }
