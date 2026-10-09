import { TOUR_ANCHORS, TOUR_ROW_STEPS, TOUR_STEPS } from './steps.ts'

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

// The steps the tour walks now: every step, less the mod-row ones while no row is on screen.
function tourShownSteps(): number[] {
  return TOUR_STEPS.flatMap((id, i) =>
    TOUR_ROW_STEPS.has(id) && resolveTourAnchor(i) === null ? [] : [i],
  )
}

/** Whether a dialog, menu or drawer is open. The tour's own popover is a Popper, not a Modal, so it never counts. */
function anyModalOpen(): boolean {
  return document.querySelector('.MuiModal-root:not(.MuiModal-hidden)') !== null
}

export { anyModalOpen, resolveTourAnchor, tourShownSteps }
