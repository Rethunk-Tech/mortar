const TOUR_STEP_COUNT = 6

const LAST_TOUR_STEP = TOUR_STEP_COUNT - 1

function clampStep(step: number): number {
  if (step < 0) {
    return 0
  }
  if (step > LAST_TOUR_STEP) {
    return LAST_TOUR_STEP
  }
  return step
}

function tourStepNext(step: number): number {
  return clampStep(step + 1)
}

function tourStepBack(step: number): number {
  return clampStep(step - 1)
}

function tourOnLastStep(step: number): boolean {
  return step >= LAST_TOUR_STEP
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

export {
  sameRectOr,
  TOUR_STEP_COUNT,
  type TourRect,
  tourEligible,
  tourOnLastStep,
  tourStepBack,
  tourStepNext,
}
