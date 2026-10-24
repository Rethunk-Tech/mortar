import { Box, Typography } from '@mui/material'
import { Check, X } from 'lucide-react'
import { type ReactNode, type RefObject, useEffect, useRef, useState } from 'react'
import type { Stage, StageId } from './stages.ts'

const TILE = 88
const RADIUS = 16
const RING = 4
const STROKE = 3
const LINE_PX = 96
const LINE_MS = 500
const LABEL_PX = 190
const BADGE = 24
const BADGE_OVERHANG = -8
const BAR_PX = 120
const ART_PX = 60
// The arc, in hundredths of the tile's outline: its length, how fast it travels and how long it takes to grow.
const ARC = 28
const PER_MS = 0.1
const GROW_MS = 350
const FULL = 100
const HALF = 2
// The tile's outline is four straight sides and four rounded corners; the left midpoint is three of each along.
const SIDES = 4
const TO_LEFT = 3
const PENDING_OPACITY = 0.35

// Where the outline's midpoints sit, in hundredths from its start at the top-left corner, clockwise: the line out of
// a tile leaves from its right midpoint, and the line into one arrives at its left midpoint.
const side = TILE + HALF * RING - STROKE
const radius = RADIUS + RING - STROKE / HALF
const straight = side - HALF * radius
const corner = (Math.PI * radius) / HALF
const perimeter = SIDES * (straight + corner)
const RIGHT = ((straight + corner + straight / HALF) / perimeter) * FULL
const LEFT = ((TO_LEFT * (straight + corner) + straight / HALF) / perimeter) * FULL

export interface StageMark {
  name: string
  art: ReactNode
  text: string
  /** The share of the stage done, 0 to 1, for a bar under it; null for none. */
  share: number | null
}

type Shown = 'pending' | 'active' | 'done' | 'failed'

const artFilter: Record<Shown, string> = {
  pending: 'grayscale(1) brightness(0.7)',
  active: 'grayscale(0.45)',
  done: 'none',
  failed: 'grayscale(0.45)',
}

const textColour: Record<Shown, string> = {
  pending: 'text.secondary',
  active: 'text.secondary',
  done: 'text.primary',
  failed: 'error.light',
}

// headAt is how far round the outline the arc's head has gone `ms` after the arc began, unwrapped: it grows out of the
// left midpoint for GROW_MS, then travels at PER_MS. Position is a function of the clock alone, so a window the system
// throttles (behind a full-screen game) still hands over on time.
function headAt(ms: number): number {
  return LEFT + ARC * Math.min(1, ms / GROW_MS) + PER_MS * Math.max(0, ms - GROW_MS)
}

// msUntil is the time after the arc began at which its head next reaches the right midpoint, from `from` ms on.
function msUntil(from: number): number {
  const ready = Math.max(from, GROW_MS)
  const ahead = (RIGHT - (headAt(ready) % FULL) + FULL) % FULL
  return ready + ahead / PER_MS
}

function paint(ring: SVGRectElement | null | undefined, ms: number) {
  if (ring) {
    const len = ARC * Math.min(1, ms / GROW_MS)
    ring.style.strokeDasharray = `${len} ${FULL - len}`
    ring.style.strokeDashoffset = `${-((headAt(ms) - len) % FULL)}`
  }
}

/**
 * useArc runs the spinner on mark `active`: it grows out of the left midpoint, where the line in arrives, and travels
 * round; once `wanted` passes the mark it runs on until its head reaches the right midpoint, where the line out
 * leaves (the last mark stops once grown), then calls done.
 */
function useArc({
  rings,
  active,
  last,
  wanted,
  done,
}: {
  rings: RefObject<(SVGRectElement | null)[]>
  active: number
  last: boolean
  wanted: RefObject<number>
  done: (i: number) => void
}) {
  useEffect(() => {
    if (active === -1) {
      return
    }
    const born = performance.now()
    let finishAt = Number.POSITIVE_INFINITY
    let frame = 0
    let timer: ReturnType<typeof setTimeout> | undefined
    const finish = () => {
      cancelAnimationFrame(frame)
      clearTimeout(timer)
      paint(rings.current[active], finishAt)
      done(active)
    }
    const step = () => {
      const ms = performance.now() - born
      if (finishAt === Number.POSITIVE_INFINITY && wanted.current > active) {
        finishAt = last ? Math.max(ms, GROW_MS) : msUntil(ms)
        // A timer as well as frames: a hidden or throttled window runs few frames.
        timer = setTimeout(finish, finishAt - ms)
      }
      if (ms >= finishAt) {
        finish()
        return
      }
      paint(rings.current[active], ms)
      frame = requestAnimationFrame(step)
    }
    frame = requestAnimationFrame(step)
    return () => {
      cancelAnimationFrame(frame)
      clearTimeout(timer)
    }
  }, [rings, active, last, wanted, done])
}

function Line({ drawn }: { drawn: boolean }) {
  return (
    <Box
      sx={{
        width: LINE_PX,
        height: STROKE,
        mt: `${(TILE - STROKE) / HALF}px`,
        bgcolor: 'var(--mortar-hairline-15)',
        borderRadius: 1,
        overflow: 'hidden',
      }}
    >
      <Box
        sx={{
          height: '100%',
          width: drawn ? '100%' : 0,
          bgcolor: 'primary.main',
          transition: `width ${LINE_MS}ms ease-in`,
        }}
      />
    </Box>
  )
}

function Ring({ on, ring }: { on: boolean; ring: (el: SVGRectElement | null) => void }) {
  return (
    <Box
      component="svg"
      aria-hidden={true}
      sx={{
        position: 'absolute',
        inset: -RING - 1,
        width: TILE + HALF * RING,
        height: TILE + HALF * RING,
        overflow: 'visible',
        opacity: on ? 1 : 0,
        transition: 'opacity 400ms',
        pointerEvents: 'none',
      }}
    >
      <Box
        component="rect"
        ref={ring}
        x={STROKE / HALF}
        y={STROKE / HALF}
        width={side}
        height={side}
        rx={radius}
        pathLength={FULL}
        sx={{
          fill: 'none',
          stroke: (t) => t.palette.primary.main,
          strokeWidth: STROKE,
          strokeLinecap: 'round',
          strokeDasharray: `0 ${FULL}`,
        }}
      />
    </Box>
  )
}

function Badge({ shown }: { shown: Shown }) {
  const failed = shown === 'failed'
  return (
    <Box
      sx={{
        position: 'absolute',
        right: BADGE_OVERHANG,
        bottom: BADGE_OVERHANG,
        width: BADGE,
        height: BADGE,
        borderRadius: '50%',
        display: 'grid',
        placeItems: 'center',
        bgcolor: failed ? 'error.main' : 'primary.main',
        color: failed ? 'error.contrastText' : 'primary.contrastText',
        transform: shown === 'done' || failed ? 'scale(1)' : 'scale(0)',
        transition: 'transform 350ms cubic-bezier(.3,1.6,.5,1)',
      }}
    >
      {failed ? <X size={14} strokeWidth={3} /> : <Check size={14} strokeWidth={3} />}
    </Box>
  )
}

function Tile({
  mark,
  shown,
  ring,
}: {
  mark: StageMark
  shown: Shown
  ring: (el: SVGRectElement | null) => void
}) {
  const edge: Record<Shown, string> = {
    pending: 'none',
    active: 'none',
    done: '0 0 0 2px var(--mortar-accent) inset',
    failed: '0 0 0 2px var(--mortar-error) inset',
  }
  return (
    <Box
      sx={{
        position: 'relative',
        width: TILE,
        height: TILE,
        p: 1.5,
        borderRadius: `${RADIUS}px`,
        display: 'grid',
        placeItems: 'center',
        bgcolor: 'var(--mortar-overlay-55)',
        border: '1px solid var(--mortar-hairline-15)',
        opacity: shown === 'pending' ? PENDING_OPACITY : 1,
        '--mortar-accent': (t) => t.palette.primary.main,
        '--mortar-error': (t) => t.palette.error.main,
        boxShadow: edge[shown],
        transition: 'opacity 600ms, box-shadow 600ms',
      }}
    >
      <Box
        sx={{
          width: ART_PX,
          height: ART_PX,
          display: 'grid',
          placeItems: 'center',
          filter: artFilter[shown],
          transition: 'filter 600ms',
          '& > *': { maxWidth: '100%', maxHeight: '100%' },
        }}
      >
        {mark.art}
      </Box>
      <Ring on={shown === 'active'} ring={ring} />
      <Badge shown={shown} />
    </Box>
  )
}

function Label({ mark, shown }: { mark: StageMark; shown: Shown }) {
  return (
    <Box
      sx={{
        position: 'absolute',
        top: '100%',
        left: '50%',
        transform: 'translateX(-50%)',
        width: LABEL_PX,
        mt: 1.25,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: 0.75,
      }}
    >
      <Typography sx={{ fontWeight: 600, fontSize: 'medium.fontSize' }} noWrap={true}>
        {mark.name}
      </Typography>
      <Typography
        noWrap={true}
        title={mark.text}
        sx={{ fontSize: 12.5, maxWidth: '100%', color: textColour[shown] }}
      >
        {mark.text}
      </Typography>
      <Box
        sx={{
          height: 4,
          width: BAR_PX,
          borderRadius: 1,
          bgcolor: 'var(--mortar-hairline-15)',
          overflow: 'hidden',
          visibility: mark.share === null ? 'hidden' : 'visible',
        }}
      >
        <Box
          sx={{
            height: '100%',
            width: `${Math.round((mark.share ?? 0) * FULL)}%`,
            bgcolor: shown === 'failed' ? 'error.main' : 'primary.main',
            transition: 'width 300ms',
          }}
        />
      </Box>
    </Box>
  )
}

/**
 * LaunchStages is the row of marks a launch passes. Each mark follows its stage, but in order and in time: the spinner
 * on the mark in progress runs on until it reaches the line to the next mark, the line draws, and only then does the
 * next mark start. A failure shows at once. onSettled is called once every mark shows done.
 */
export function LaunchStages({
  stages,
  marks,
  onSettled,
}: {
  stages: Stage[]
  marks: Record<StageId, StageMark>
  onSettled: () => void
}) {
  const firstOpen = stages.findIndex((s) => s.state !== 'done')
  const failed = stages.findIndex((s) => s.state === 'failed')
  const [shown, setShown] = useState(0)
  const [lines, setLines] = useState(0)
  const rings = useRef<(SVGRectElement | null)[]>([])
  const wanted = useRef(0)
  wanted.current = firstOpen === -1 ? stages.length : firstOpen
  const active = failed === -1 && shown < stages.length && lines >= shown ? shown : -1
  const [done] = useState(() => (i: number) => setShown(i + 1))
  useArc({ rings, active, last: active === stages.length - 1, wanted, done })

  // The line out of a mark draws once the mark shows done; the next mark starts when it arrives.
  useEffect(() => {
    if (shown > lines && shown < stages.length) {
      const timer = setTimeout(() => setLines(shown), LINE_MS)
      return () => clearTimeout(timer)
    }
    if (shown === stages.length && stages.length > 0) {
      onSettled()
    }
    return
  }, [shown, lines, stages.length, onSettled])

  const stateOf = (i: number): Shown => {
    if (i === failed) {
      return 'failed'
    }
    if (i < shown || (failed !== -1 && i < failed)) {
      return 'done'
    }
    return i === active ? 'active' : 'pending'
  }
  return (
    <Box sx={{ display: 'flex', alignItems: 'flex-start', mb: '84px' }}>
      {stages.map((stage, i) => (
        <Box key={stage.id} sx={{ display: 'flex', alignItems: 'flex-start' }}>
          {i > 0 ? <Line drawn={lines >= i || (failed !== -1 && i <= failed)} /> : null}
          <Box sx={{ position: 'relative' }} data-stage={stage.id} data-state={stateOf(i)}>
            <Tile
              mark={marks[stage.id]}
              shown={stateOf(i)}
              ring={(el) => {
                rings.current[i] = el
              }}
            />
            <Label mark={marks[stage.id]} shown={stateOf(i)} />
          </Box>
        </Box>
      ))}
    </Box>
  )
}
