import type { PlayPreset } from '../profiles/profilePresets.ts'

type PlayMenuEntry = { kind: 'play'; key: string } | { kind: 'setDefault' } | { kind: 'vanilla' }

/** The Play menu's rows in focus order; every row is one MenuItem, so arrow keys visit each exactly once. */
function playMenuEntries(presets: PlayPreset[]): PlayMenuEntry[] {
  return [
    ...presets.map((p): PlayMenuEntry => ({ kind: 'play', key: p.key })),
    ...(presets.length > 0 ? [{ kind: 'setDefault' } as const] : []),
    { kind: 'vanilla' },
  ]
}

export { playMenuEntries }
