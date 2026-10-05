type Mode = 'off' | 'gray' | 'hide'

// What Browse does with mods in the open profile, marked obsolete, or marked broken; the extension's three rows.
interface BrowseModes {
  installed: Mode
  obsolete: Mode
  broken: Mode
}

const OFF: BrowseModes = { installed: 'off', obsolete: 'off', broken: 'off' }
const ROWS = ['installed', 'obsolete', 'broken'] as const
const isMode = (v: string): v is Mode => v === 'off' || v === 'gray' || v === 'hide'

// The setting is "installed=gray obsolete=hide"; anything unreadable is Off.
function parseModes(saved: string): BrowseModes {
  const out = { ...OFF }
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
  return ROWS.filter((r) => modes[r] !== 'off')
    .map((r) => `${r}=${modes[r]}`)
    .join(' ')
}

export type { BrowseModes, Mode }
export { formatModes, OFF, parseModes, ROWS }
