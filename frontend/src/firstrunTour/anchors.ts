import { TOUR_ANCHORS, TOUR_STEPS } from './steps.ts'

function resolveTourAnchor(step: number): HTMLElement | null {
  const id = TOUR_STEPS[step]
  for (const selector of id ? TOUR_ANCHORS[id] : []) {
    const el = document.querySelector<HTMLElement>(selector)
    if (el) {
      return el
    }
  }
  return null
}

export { resolveTourAnchor }
