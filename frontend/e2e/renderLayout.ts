import type { Page } from '@playwright/test'

export interface LayoutBreak {
  kind: 'collapsed' | 'covered' | 'overflow'
  where: string
  text: string
}

// Layout breaks no baseline is needed to see: a picture or control squeezed to a sliver, the start of a text covered by
// another element, and a page wider than its window. Each is a bug on any screen.
// The checks run in the page, so each carries its own helpers.
export async function layoutBreaks(page: Page): Promise<LayoutBreak[]> {
  return [
    ...(await page.evaluate(collapsed)),
    ...(await page.evaluate(covered)),
    ...(await page.evaluate(overflow)),
  ]
}

export function collapsed(): LayoutBreak[] {
  const sliverPx = 8
  const tallPx = 16
  const out: LayoutBreak[] = []
  const layers = [...document.querySelectorAll('[role="dialog"], [role="menu"], [role="listbox"]')]
  const top = layers.filter((el) => el.getBoundingClientRect().width > 0).at(-1) ?? document.body
  for (const el of top.querySelectorAll('img, [role="img"], button, [role="button"], a[href]')) {
    const r = el.getBoundingClientRect()
    const inView = r.bottom > 0 && r.top < innerHeight && r.right > 0 && r.left < innerWidth
    const thin = Math.min(r.width, r.height)
    if (inView && thin < sliverPx && Math.max(r.width, r.height) >= tallPx) {
      const name = el.getAttribute('aria-label') ?? el.getAttribute('alt') ?? el.textContent ?? ''
      out.push({
        kind: 'collapsed',
        where: `${el.parentElement?.tagName.toLowerCase() ?? ''}>${el.tagName.toLowerCase()}`,
        text: `${Math.round(r.width)}x${Math.round(r.height)} ${name.trim().slice(0, 60)}`,
      })
    }
  }
  return out
}

// The first letter of each visible text: a sticky or fixed bar scrolled over it is normal, anything else on top is not.
export function covered(): LayoutBreak[] {
  const pinned = (el: Element | null): boolean =>
    el !== null && (/sticky|fixed/.test(getComputedStyle(el).position) || pinned(el.parentElement))
  // What every clipping ancestor leaves visible: text scrolled out of its list is clipped, not covered.
  const visible = (el: Element | null): DOMRect => {
    let left = 0
    let top = 0
    let right = innerWidth
    let bottom = innerHeight
    for (let at = el; at; at = at.parentElement) {
      const s = getComputedStyle(at)
      if (/auto|scroll|hidden|clip/.test(`${s.overflowX} ${s.overflowY}`)) {
        const b = at.getBoundingClientRect()
        left = Math.max(left, b.left)
        top = Math.max(top, b.top)
        right = Math.min(right, b.right)
        bottom = Math.min(bottom, b.bottom)
      }
    }
    return new DOMRect(left, top, right - left, bottom - top)
  }
  // A transparent ::before or ::after stretched past its element makes a whole card one click target (the title's, on a
  // mod card); the text under it still shows.
  const transparent = (el: Element, pseudo: string) =>
    /rgba\(0, 0, 0, 0\)|transparent/.test(getComputedStyle(el, pseudo).backgroundColor)
  const clickArea = (el: Element, x: number, y: number) => {
    const b = el.getBoundingClientRect()
    const outside = x < b.left || x > b.right || y < b.top || y > b.bottom
    return outside && transparent(el, '::before') && transparent(el, '::after')
  }
  const firstLetter = (node: Text): DOMRect | null => {
    const at = (node.textContent ?? '').search(/\S/)
    if (at < 0 || (node.textContent ?? '').trim().length < 2) {
      return null
    }
    const range = document.createRange()
    range.setStart(node, at)
    range.setEnd(node, at + 1)
    const r = range.getBoundingClientRect()
    const view = visible(node.parentElement)
    const inside =
      r.top >= view.top && r.bottom <= view.bottom && r.left >= view.left && r.right <= view.right
    return r.width > 0 && inside ? r : null
  }
  const layers = [...document.querySelectorAll('[role="dialog"], [role="menu"], [role="listbox"]')]
  const top = layers.filter((el) => el.getBoundingClientRect().width > 0).at(-1) ?? document.body
  const out: LayoutBreak[] = []
  const walker = document.createTreeWalker(top, NodeFilter.SHOW_TEXT)
  for (let node = walker.nextNode(); node; node = walker.nextNode()) {
    const owner = node.parentElement
    // Text kept for screen readers sits in a 1px clipped box, and a hit test passes through text that takes no pointer
    // events (a field's floating label), so neither can be judged here.
    const box = owner?.getBoundingClientRect()
    const style = owner ? getComputedStyle(owner) : null
    if (!(box && style) || box.width <= 1 || box.height <= 1 || style.pointerEvents === 'none') {
      continue
    }
    const r = style.visibility !== 'hidden' && node instanceof Text ? firstLetter(node) : null
    const x = r ? r.left + r.width / 2 : 0
    const y = r ? r.top + r.height / 2 : 0
    const hit = r ? document.elementFromPoint(x, y) : null
    if (
      !(owner && hit) ||
      hit === owner ||
      owner.contains(hit) ||
      hit.contains(owner) ||
      (pinned(hit) && !pinned(owner)) ||
      // Notifications float over whatever is under them for a few seconds, a sticky panel included.
      hit.closest('[aria-live]') !== null ||
      clickArea(hit, x, y)
    ) {
      continue
    }
    const name = (hit.getAttribute('aria-label') ?? hit.tagName).toLowerCase()
    out.push({
      kind: 'covered',
      where: `${owner.parentElement?.tagName.toLowerCase() ?? ''}>${owner.tagName.toLowerCase()}`,
      text: `"${(node.textContent ?? '').trim().slice(0, 40)}" under ${name}`,
    })
  }
  return out
}

export function overflow(): LayoutBreak[] {
  const root = document.documentElement
  return root.scrollWidth > root.clientWidth + 1
    ? [
        {
          kind: 'overflow',
          where: 'html',
          text: `${root.scrollWidth}px wide in a ${root.clientWidth}px window`,
        },
      ]
    : []
}
