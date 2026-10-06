// shown is the ascending list of step indexes the tour walks; a step left out is passed over both ways.
function tourStepNext(step: number, shown: number[]): number {
  return shown.find((i) => i > step) ?? step
}

function tourStepBack(step: number, shown: number[]): number {
  return shown.findLast((i) => i < step) ?? step
}

function tourOnLastStep(step: number, shown: number[]): boolean {
  return step >= (shown.at(-1) ?? 0)
}

interface TourRect {
  top: number
  left: number
  width: number
  height: number
}

// Keeps the previous object when nothing moved, so the 250 ms anchor poll does not re-render.
function sameRectOr(prev: TourRect | null, next: TourRect | null): TourRect | null {
  if (!next) {
    return null
  }
  if (
    prev &&
    prev.top === next.top &&
    prev.left === next.left &&
    prev.width === next.width &&
    prev.height === next.height
  ) {
    return prev
  }
  return { top: next.top, left: next.left, width: next.width, height: next.height }
}

/** Whether the tour should open: on a game with an open profile, unseen and not closed this session, or replayed.
 * A close counts at once, so a failed save of the seen flag cannot reopen it. */
function tourEligible(o: {
  onGameWithProfile: boolean
  unseen: boolean
  dismissed: boolean
  replay: boolean
}): boolean {
  return o.onGameWithProfile && ((o.unseen && !o.dismissed) || o.replay)
}

export { sameRectOr, type TourRect, tourEligible, tourOnLastStep, tourStepBack, tourStepNext }
