import type { Page } from '@playwright/test'

/** A boxed element (an edge or a fill with rounded corners) that sits inside another one on the screen. Dialog, menu
 * and popover papers are the surface their content sits on, not a card, so they do not count as the outer box. */
export interface NestedBox {
  inner: string
  outer: string
  text: string
}

export function nestedBoxes(page: Page): Promise<NestedBox[]> {
  return page.evaluate(() => {
    const Surface = '.MuiDialog-paper, .MuiPopover-paper, .MuiMenu-paper, .MuiTooltip-tooltip'
    const clear = (c: string) => c === 'rgba(0, 0, 0, 0)' || c === 'transparent'
    const boxed = (el: Element): boolean => {
      const s = getComputedStyle(el)
      const radius = Number.parseFloat(s.borderTopLeftRadius) > 0
      const edge = ['Top', 'Right', 'Bottom', 'Left'].every(
        (side) =>
          Number.parseFloat(s.getPropertyValue(`border-${side.toLowerCase()}-width`)) > 0 &&
          !clear(s.getPropertyValue(`border-${side.toLowerCase()}-color`)),
      )
      const box = el.getBoundingClientRect()
      const big = box.width >= 120 && box.height >= 28
      return radius && big && (edge || !clear(s.backgroundColor))
    }
    const name = (el: Element) => {
      const aria = el.getAttribute('aria-label')
      const cls = [...el.classList]
        .filter((c) => c.startsWith('Mui'))
        .slice(0, 2)
        .join('.')
      return `${el.tagName.toLowerCase()}${cls ? `.${cls}` : ''}${aria ? `[${aria}]` : ''}`
    }
    const out: NestedBox[] = []
    for (const el of document.querySelectorAll('body *')) {
      const r = el.getBoundingClientRect()
      if (
        r.width === 0 ||
        r.height === 0 ||
        el.closest('.MuiButtonBase-root, .MuiChip-root, input, textarea')
      ) {
        continue
      }
      if (!boxed(el)) {
        continue
      }
      let up = el.parentElement
      while (up && up !== document.body) {
        if (up.matches(Surface)) {
          break
        }
        if (boxed(up)) {
          out.push({
            inner: name(el),
            outer: name(up),
            text: (el.textContent ?? '').trim().slice(0, 60),
          })
          break
        }
        up = up.parentElement
      }
    }
    return out
  })
}
