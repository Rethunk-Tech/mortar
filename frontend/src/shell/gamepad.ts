import { runShortcut } from '../settings/useShortcuts.ts'
import { FOCUSABLE } from './arrowFocus.ts'
import {
  type Direction,
  firedActions,
  heldActions,
  type PadAction,
  spatialNext,
} from './gamepadInput.ts'

const ARROWS: Record<Direction, string> = {
  up: 'ArrowUp',
  down: 'ArrowDown',
  left: 'ArrowLeft',
  right: 'ArrowRight',
}

/** The open dialog, menu or drawer that holds focus, or the page when none is open. */
function topLayer(): HTMLElement {
  return (
    [...document.querySelectorAll<HTMLElement>('.MuiModal-root:not(.MuiModal-hidden)')].at(-1) ??
    document.body
  )
}

function reachable(el: HTMLElement): boolean {
  return (
    el.tabIndex >= 0 &&
    !el.matches(':disabled') &&
    el.getClientRects().length > 0 &&
    el.closest('[aria-hidden="true"], [inert]') === null
  )
}

// key sends the keydown a keyboard would, so the focused control's own handler (a roving list, tabs, a menu, an
// arrowFocus grid, a dialog's Esc) acts on it; it reports whether no handler took it.
function key(target: EventTarget, name: string): boolean {
  return target.dispatchEvent(
    new KeyboardEvent('keydown', { key: name, bubbles: true, cancelable: true }),
  )
}

function focused(): HTMLElement | null {
  const el = document.activeElement
  return el instanceof HTMLElement && el !== document.body ? el : null
}

function move(dir: Direction) {
  const el = focused()
  const scope = topLayer()
  if (el && scope.contains(el)) {
    const free = key(el, ARROWS[dir])
    // A control that keeps the arrows for its value or its popup has had them; a list at its end lets focus leave.
    if (!free && (focused() !== el || el.matches('[role="slider"], [aria-haspopup]'))) {
      return
    }
  }
  const reached = [...scope.querySelectorAll<HTMLElement>(FOCUSABLE)].filter(reachable)
  // A focusable scroll area is a stop for the keyboard's scrolling; the pad goes on into the controls it holds.
  const items = reached.filter(
    (item) => !reached.some((other) => other !== item && item.contains(other)),
  )
  const inside = el ? items.find((item) => item !== el && el.contains(item)) : undefined
  const at = el ? items.indexOf(el) : -1
  const next =
    el && scope.contains(el) && !inside
      ? items[
          spatialNext(
            el.getBoundingClientRect(),
            items.map((item) => item.getBoundingClientRect()),
            dir,
            at,
          )
        ]
      : (inside ?? items[0])
  if (next) {
    next.focus()
    next.scrollIntoView({ block: 'nearest', inline: 'nearest' })
  }
}

function switchTab(step: number) {
  const tabs = [...document.querySelectorAll<HTMLElement>('[data-profile-tabs] [role="tab"]')]
  const at = tabs.findIndex((tab) => tab.getAttribute('aria-selected') === 'true')
  const next = tabs[(at + step + tabs.length) % tabs.length]
  next?.click()
  next?.focus()
}

function act(action: PadAction) {
  // The browser shows a focus ring only after keyboard use; this marks focus moved by the pad for the theme's ring.
  document.documentElement.dataset.input = 'gamepad'
  const modal = topLayer() !== document.body
  switch (action) {
    case 'confirm': {
      const el = focused()
      if (el && key(el, 'Enter')) {
        el.click()
      }
      return
    }
    case 'back':
      key(focused() ?? document.body, 'Escape')
      return
    case 'play':
      if (!modal) {
        runShortcut('play')
      }
      return
    case 'prevTab':
    case 'nextTab':
      if (!modal) {
        switchTab(action === 'prevTab' ? -1 : 1)
      }
      return
    default:
      move(action)
  }
}

let polling = false
const due = new Map<PadAction, number>()

function poll(now: number) {
  const pads = navigator.getGamepads()
  if (!pads.some((pad) => pad?.connected)) {
    polling = false
    due.clear()
    return
  }
  for (const action of firedActions(heldActions(pads), due, now)) {
    act(action)
  }
  requestAnimationFrame(poll)
}

function startPolling() {
  if (!polling) {
    polling = true
    requestAnimationFrame(poll)
  }
}

// The Gamepad API is polled, so frames are read only while a pad is connected; a pad shows up after its first press.
export function initGamepad() {
  if (typeof navigator.getGamepads !== 'function') {
    return
  }
  addEventListener('gamepadconnected', startPolling)
  addEventListener(
    'pointerdown',
    () => {
      delete document.documentElement.dataset.input
    },
    true,
  )
  if (navigator.getGamepads().some((pad) => pad?.connected)) {
    startPolling()
  }
}
