const TOUR_STEPS = [
  'profiles',
  'browse',
  'game',
  'play',
  'mods',
  'details',
  'config',
  'problems',
  'command',
] as const

type TourStep = (typeof TOUR_STEPS)[number]

// What each step spotlights: the first selector that matches wins.
const MOD_ROW = ['main [data-mod-row]', 'main [data-mod-id]']
const TOUR_ANCHORS: Record<TourStep, string[]> = {
  profiles: ['[data-tour="profile-switcher"]'],
  browse: ['[data-tour="browse-tab"]'],
  game: ['[data-tour="game-tab"]'],
  play: ['main nav .MuiButtonGroup-root button', 'main nav button.MuiButton-contained', 'main nav'],
  mods: ['[data-tour="mods-tab"]'],
  details: MOD_ROW,
  config: MOD_ROW,
  problems: ['[data-tour="problems-tab"]'],
  command: ['[data-tour="app-menu"]'],
}

// Steps about a mod row, left out while the list shows none: there is nothing on screen for them to point at.
const TOUR_ROW_STEPS: ReadonlySet<TourStep> = new Set(['details', 'config'])

export { TOUR_ANCHORS, TOUR_ROW_STEPS, TOUR_STEPS }
