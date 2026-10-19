import React from 'react'
import { SystemAccent } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { type AccentName, accents, defaultAccent } from '../theme/accents.ts'

export const SYSTEM_ACCENT = 'system'

export function isAccent(value: string): value is AccentName {
  return value in accents
}

/** The accent colour for a stored accent choice; "system" takes the desktop's colour when it gave one. */
export function accentColour(accent: string, system: string): string {
  if (accent === SYSTEM_ACCENT && system !== '') {
    return system
  }
  return accents[isAccent(accent) ? accent : defaultAccent]
}

let systemAccent = ''
let systemAccentRead: Promise<void> | undefined

/** The desktop's accent colour as "#rrggbb", empty until read or when the desktop has none. Read once per run; a screen
 * that mounts later starts with the answer instead of re-rendering when it arrives. */
export function useSystemAccent(): string {
  const [colour, setColour] = React.useState(systemAccent)
  React.useEffect(() => {
    systemAccentRead ??= SystemAccent()
      .then((c) => {
        systemAccent = c
      })
      .catch(() => {
        systemAccent = ''
      })
    systemAccentRead.then(() => setColour(systemAccent))
  }, [])
  return colour
}
