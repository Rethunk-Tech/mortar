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

// What each step spotlights: the first selector that matches wins, so a step falls back to a control that is always
// on screen when the list it prefers is empty or on another tab.
const MOD_ROW = ['main [data-mod-row]', 'main [data-mod-id]', '[data-tour="mods-tab"]']
const TOUR_ANCHORS: Record<TourStep, string[]> = {
  profiles: ['main nav'],
  browse: ['[data-tour="browse-tab"]'],
  game: ['[data-tour="game-tab"]'],
  play: ['main nav .MuiButtonGroup-root button', 'main nav button.MuiButton-contained', 'main nav'],
  mods: ['[data-tour="mods-tab"]'],
  details: MOD_ROW,
  config: MOD_ROW,
  problems: ['[data-tour="problems-tab"]'],
  command: ['[data-tour="app-menu"]'],
}

export { TOUR_ANCHORS, TOUR_STEPS }
