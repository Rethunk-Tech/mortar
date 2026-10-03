import {
  TOUR_STEP_COMMAND,
  TOUR_STEP_MODS,
  TOUR_STEP_PLAY,
  TOUR_STEP_PROBLEMS,
  TOUR_STEP_PROFILES,
} from './steps.ts'

const MODS_TAB_INDEX = 0
const PROBLEMS_TAB_INDEX = 1

function mainNav(): HTMLElement | null {
  return document.querySelector('main nav')
}

function workspaceTabButtons(): HTMLElement[] {
  const tablist = document.querySelector('main [role="tablist"]')
  if (!tablist) {
    return []
  }
  return [...tablist.querySelectorAll<HTMLElement>('[role="tab"]')]
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
    case TOUR_STEP_MODS: {
      const tabs = workspaceTabButtons()
      return tabs[MODS_TAB_INDEX] ?? null
    }
    case TOUR_STEP_PROBLEMS: {
      const tabs = workspaceTabButtons()
      return tabs[PROBLEMS_TAB_INDEX] ?? null
    }
    case TOUR_STEP_COMMAND:
      return document.querySelector('main')
    default:
      return null
  }
}

export { resolveTourAnchor }
