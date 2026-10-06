type Mode = 'off' | 'gray' | 'hide'

// What Browse does with mods in the open profile, marked obsolete, or marked broken; the extension's three rows.
interface BrowseModes {
  installed: Mode
  obsolete: Mode
  broken: Mode
}

// Hidden unless the player chooses otherwise: Browse is for finding mods to add, and these are not.
const DEFAULT_MODES: BrowseModes = { installed: 'hide', obsolete: 'hide', broken: 'hide' }
const ROWS = ['installed', 'obsolete', 'broken'] as const
const isMode = (v: string): v is Mode => v === 'off' || v === 'gray' || v === 'hide'

// The setting is "installed=gray obsolete=off broken=hide"; a row it does not name keeps its default.
function parseModes(saved: string): BrowseModes {
  const out = { ...DEFAULT_MODES }
  for (const pair of saved.split(' ')) {
    const [row, mode = ''] = pair.split('=')
    const known = ROWS.find((r) => r === row)
    if (known && isMode(mode)) {
      out[known] = mode
    }
  }
  return out
}

function formatModes(modes: BrowseModes): string {
  return ROWS.map((r) => `${r}=${modes[r]}`).join(' ')
}

export type { BrowseModes, Mode }
export { DEFAULT_MODES, formatModes, parseModes, ROWS }
