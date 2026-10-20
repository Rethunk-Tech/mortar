import type { Page } from '@playwright/test'

/** A stack whose rows sit unevenly apart, or a card whose content is inset more on one side than the other. Distances
 * are between what paints (text, fills, edges, pictures), so padding hidden in a wrapper still counts. */
export interface UnevenSpace {
  kind: 'gaps' | 'inset'
  where: string
  sizes: number[]
  text: string
}

export function unevenSpacing(page: Page): Promise<UnevenSpace[]> {
  return page.evaluate(() => {
    const TolerancePx = 2
    const MaxPainted = 400
    const clear = (c: string) => c === 'rgba(0, 0, 0, 0)' || c === 'transparent'
    const shown = (r: DOMRect) =>
      r.width > 0 &&
      r.height > 0 &&
      r.bottom > 0 &&
      r.top < innerHeight &&
      r.right > 0 &&
      r.left < innerWidth
    const paints = (el: Element): boolean => {
      if (el.matches('img, svg, input, textarea, canvas, video')) {
        return true
      }
      const s = getComputedStyle(el)
      return (
        !clear(s.backgroundColor) ||
        (Number.parseFloat(s.borderTopWidth) > 0 && !clear(s.borderTopColor))
      )
    }
    // The box around everything inside el that paints: its own fill or edge, text, pictures.
    const painted = (el: Element): DOMRect | null => {
      let top = Number.POSITIVE_INFINITY
      let left = top
      let bottom = Number.NEGATIVE_INFINITY
      let right = bottom
      let n = 0
      const add = (r: DOMRect) => {
        if (shown(r)) {
          top = Math.min(top, r.top)
          left = Math.min(left, r.left)
          bottom = Math.max(bottom, r.bottom)
          right = Math.max(right, r.right)
        }
      }
      const walk = document.createTreeWalker(el, NodeFilter.SHOW_ELEMENT | NodeFilter.SHOW_TEXT)
      for (
        let node: Node | null = walk.currentNode;
        node && n < MaxPainted;
        node = walk.nextNode()
      ) {
        n++
        if (node.nodeType === Node.TEXT_NODE) {
          if ((node.textContent ?? '').trim() !== '') {
            const range = document.createRange()
            range.selectNodeContents(node)
            add(range.getBoundingClientRect())
          }
        } else if (
          getComputedStyle(node as Element).visibility !== 'hidden' &&
          paints(node as Element)
        ) {
          add((node as Element).getBoundingClientRect())
        }
      }
      return Number.isFinite(top) ? new DOMRect(left, top, right - left, bottom - top) : null
    }
    const name = (el: Element) => {
      const aria = el.getAttribute('aria-label')
      const cls = [...el.classList]
        .filter((c) => c.startsWith('Mui'))
        .slice(0, 2)
        .join('.')
      return `${el.tagName.toLowerCase()}${cls ? `.${cls}` : ''}${aria ? `[${aria}]` : ''}`
    }
    const uneven = (sizes: number[]) => Math.max(...sizes) - Math.min(...sizes) > TolerancePx
    const finding = (kind: UnevenSpace['kind'], el: Element, sizes: number[]): UnevenSpace => ({
      kind,
      where: name(el),
      sizes,
      text: (el.textContent ?? '').trim().slice(0, 60),
    })
    // A stack: three or more rows, each wholly below the last.
    const stackGaps = (el: Element): UnevenSpace | null => {
      const rows = [...el.children]
        .filter((c) => !['absolute', 'fixed'].includes(getComputedStyle(c).position))
        .map((c) => painted(c))
        .filter((r): r is DOMRect => r !== null)
      const stacked = rows.every((r, i) => i === 0 || r.top >= (rows[i - 1]?.bottom ?? 0) - 1)
      if (rows.length < 3 || !stacked) {
        return null
      }
      const gaps = rows.slice(1).map((r, i) => Math.round(r.top - (rows[i]?.bottom ?? 0)))
      return uneven(gaps) ? finding('gaps', el, gaps) : null
    }
    // A card: a filled or edged box whose content is inset more above than below, or the other way.
    const cardInset = (el: Element, box: DOMRect): UnevenSpace | null => {
      if (
        !paints(el) ||
        el.matches('img, svg, input, textarea, canvas, video') ||
        box.height < 40
      ) {
        return null
      }
      const inner = [...el.children].map((c) => painted(c)).filter((r): r is DOMRect => r !== null)
      if (inner.length === 0) {
        return null
      }
      const top = Math.round(Math.min(...inner.map((r) => r.top)) - box.top)
      const bottom = Math.round(box.bottom - Math.max(...inner.map((r) => r.bottom)))
      return top > 0 && bottom > 0 && uneven([top, bottom])
        ? finding('inset', el, [top, bottom])
        : null
    }
    const out: UnevenSpace[] = []
    for (const el of document.querySelectorAll('body *')) {
      const box = el.getBoundingClientRect()
      if (
        shown(box) &&
        !el.closest('.MuiButtonBase-root, .MuiChip-root, .MuiInputBase-root, svg')
      ) {
        for (const f of [stackGaps(el), cardInset(el, box)]) {
          if (f) {
            out.push(f)
          }
        }
      }
    }
    return out
  })
}
