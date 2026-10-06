type Direction = 'up' | 'down' | 'left' | 'right'
type PadAction = Direction | 'confirm' | 'back' | 'prevTab' | 'nextTab' | 'play'

interface Rect {
  top: number
  bottom: number
  left: number
  right: number
}

// The standard mapping (w3c.github.io/gamepad/#remapping), which Steam Input's virtual pad and an Xbox-style pad report.
const BUTTON = {
  a: 0,
  b: 1,
  lb: 4,
  rb: 5,
  start: 9,
  up: 12,
  down: 13,
  left: 14,
  right: 15,
} as const
const BUTTONS: readonly (readonly [number, PadAction])[] = [
  [BUTTON.a, 'confirm'],
  [BUTTON.b, 'back'],
  [BUTTON.lb, 'prevTab'],
  [BUTTON.rb, 'nextTab'],
  [BUTTON.start, 'play'],
  [BUTTON.up, 'up'],
  [BUTTON.down, 'down'],
  [BUTTON.left, 'left'],
  [BUTTON.right, 'right'],
]
const STICK = 0.5
const REPEAT_DELAY_MS = 400
const REPEAT_MS = 120
const DIRECTIONS: ReadonlySet<PadAction> = new Set(['up', 'down', 'left', 'right'])

/** The actions held on every pad, the left stick read as the d-pad. */
function heldActions(pads: readonly (Pick<Gamepad, 'buttons' | 'axes'> | null)[]): Set<PadAction> {
  const held = new Set<PadAction>()
  for (const pad of pads.filter((p) => p !== null)) {
    for (const [i, action] of BUTTONS) {
      if (pad.buttons[i]?.pressed) {
        held.add(action)
      }
    }
    const [x = 0, y = 0] = pad.axes
    if (x <= -STICK) {
      held.add('left')
    } else if (x >= STICK) {
      held.add('right')
    }
    if (y <= -STICK) {
      held.add('up')
    } else if (y >= STICK) {
      held.add('down')
    }
  }
  return held
}

/**
 * The actions this frame fires: each one as it is pressed, and a held direction again after a pause and then at a
 * steady rate, so holding the stick walks a list. due keeps when each held direction fires next.
 */
function firedActions(held: ReadonlySet<PadAction>, due: Map<PadAction, number>, now: number) {
  const fired: PadAction[] = []
  for (const action of due.keys()) {
    if (!held.has(action)) {
      due.delete(action)
    }
  }
  for (const action of held) {
    const at = due.get(action)
    if (at === undefined) {
      fired.push(action)
      due.set(action, DIRECTIONS.has(action) ? now + REPEAT_DELAY_MS : Number.POSITIVE_INFINITY)
    } else if (now >= at) {
      fired.push(action)
      due.set(action, now + REPEAT_MS)
    }
  }
  return fired
}

const SLACK_PX = 4
const ORTHO_WEIGHT = 2

/**
 * spatialNext is the rect a direction moves to from `from`: one whose centre lies that way, scored by the gap along
 * the direction plus twice the sideways gap, so the nearest control in line wins over a nearer one off to the side; a tie goes to the most centred.
 * skip is from's own index in rects, or -1. -1 means nothing lies that way.
 */
function spatialNext(from: Rect, rects: readonly Rect[], dir: Direction, skip = -1): number {
  const cx = (r: Rect) => (r.left + r.right) / 2
  const cy = (r: Rect) => (r.top + r.bottom) / 2
  const horizontal = dir === 'left' || dir === 'right'
  const sign = dir === 'right' || dir === 'down' ? 1 : -1
  let best = -1
  let bestScore = Number.POSITIVE_INFINITY
  let bestAside = Number.POSITIVE_INFINITY
  rects.forEach((r, i) => {
    if (i === skip) {
      return
    }
    const along = horizontal ? (cx(r) - cx(from)) * sign : (cy(r) - cy(from)) * sign
    if (along <= SLACK_PX) {
      return
    }
    const ahead = horizontal
      ? [r.left - from.right, from.left - r.right]
      : [r.top - from.bottom, from.top - r.bottom]
    const lead = (sign > 0 ? ahead[0] : ahead[1]) ?? 0
    // One that overlaps from along the way (a control inside a wide row) counts only when it lies more ahead than aside.
    const aside = horizontal ? Math.abs(cy(r) - cy(from)) : Math.abs(cx(r) - cx(from))
    if (lead < 0 && aside >= along) {
      return
    }
    const gap = Math.max(0, lead)
    const side = horizontal
      ? Math.max(0, r.top - from.bottom, from.top - r.bottom)
      : Math.max(0, r.left - from.right, from.left - r.right)
    const score = gap + ORTHO_WEIGHT * side
    if (score < bestScore || (score === bestScore && aside < bestAside)) {
      best = i
      bestScore = score
      bestAside = aside
    }
  })
  return best
}

export type { Direction, PadAction }
export { firedActions, heldActions, spatialNext }
