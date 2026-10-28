import { ThemeProvider, useMediaQuery } from '@mui/material'
import { Window } from '@wailsio/runtime'
import React from 'react'
import { useSettings } from './settings/store.ts'
import { accentColour, isAccent, useSystemAccent } from './settings/theme.ts'
import { defaultAccent } from './theme/accents.ts'
import { honourTheme } from './theme/palette.ts'
import { useReducedMotion } from './theme/reducedMotion.ts'
import { useRoomy } from './theme/roomy.ts'
import { createMortarTheme } from './theme/theme.ts'

const DEFAULT_UI_SCALE = 100
const PERCENT = 100

export function Themed({ children }: { children: React.ReactNode }) {
  const accent = useSettings((s) => s.accent)
  const systemAccent = useSystemAccent()
  const font = useSettings((s) => s.font)
  const density = useSettings((s) => s.density)
  const roomy = useRoomy((s) => s.roomy)
  const reduceMotion = useReducedMotion()
  const touchRipples = useSettings((s) => s.touchRipples)
  const colourTheme = useSettings((s) => s.theme)
  const uiScale = useSettings((s) => s.uiScale)
  // The webview's own zoom scales every size alike, as a browser's page zoom does; a page served to a browser has no
  // window to zoom, so CSS zoom stands in there.
  React.useEffect(() => {
    const zoom = (uiScale || DEFAULT_UI_SCALE) / PERCENT
    Window.SetZoom(zoom).catch(() => {
      document.documentElement.style.zoom = String(zoom)
    })
  }, [uiScale])
  const osLight = useMediaQuery('(prefers-color-scheme: light)', { noSsr: true })
  const theme = React.useMemo(
    () =>
      createMortarTheme(isAccent(accent) ? accent : defaultAccent, {
        font,
        accentHex: accentColour(accent, systemAccent),
        compact: density === 'compact' && !roomy,
        roomy,
        reduceMotion,
        ripples: touchRipples === true,
        mode: honourTheme(colourTheme || 'dark', osLight),
      }),
    [accent, font, systemAccent, colourTheme, density, osLight, reduceMotion, roomy, touchRipples],
  )
  return <ThemeProvider theme={theme}>{children}</ThemeProvider>
}
