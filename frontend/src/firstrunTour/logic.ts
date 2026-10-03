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

export { TOUR_STEP_COUNT, tourOnLastStep, tourStepBack, tourStepNext }
