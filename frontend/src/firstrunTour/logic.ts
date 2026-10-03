const TOUR_STEP_COUNT = 5

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

export { sameRectOr, TOUR_STEP_COUNT, type TourRect, tourOnLastStep, tourStepBack, tourStepNext }
