import {
  TOUR_STEP_COMMAND,
  TOUR_STEP_MODS,
  TOUR_STEP_PLAY,
  TOUR_STEP_PROBLEMS,
  TOUR_STEP_PROFILES,
} from './steps.ts'

function mainNav(): HTMLElement | null {
  return document.querySelector('main nav')
}

function tourTarget(id: string): HTMLElement | null {
  return document.querySelector<HTMLElement>(`main [data-tour="${id}"]`)
}

function resolveTourAnchor(step: number): HTMLElement | null {
  const nav = mainNav()
  switch (step) {
    case TOUR_STEP_PROFILES:
      return nav
    case TOUR_STEP_PLAY: {
      const play =
        nav?.querySelector<HTMLElement>('.MuiButtonGroup-root button') ??
        nav?.querySelector<HTMLElement>('button.MuiButton-contained')
      return play ?? nav
    }
    case TOUR_STEP_MODS:
      return tourTarget('mods-tab')
    case TOUR_STEP_PROBLEMS:
      return tourTarget('problems-tab')
    case TOUR_STEP_COMMAND:
      return document.querySelector<HTMLElement>('[data-tour="app-menu"]')
    default:
      return null
  }
}

export { resolveTourAnchor }
