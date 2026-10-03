import { ThemeProvider } from '@mui/material'
import React from 'react'
import { useSettings } from './settings/store.ts'
import { isAccent } from './settings/theme.ts'
import { defaultAccent } from './theme/accents.ts'
import { honourTheme } from './theme/palette.ts'
import { createMortarTheme } from './theme/theme.ts'

function honourReducedMotion(mode: string, os: boolean) {
  if (mode === 'always') {
    return true
  }
  if (mode === 'never') {
    return false
  }
  return os
}

export function Themed({ children }: { children: React.ReactNode }) {
  const accent = useSettings((s) => s.accent)
  const density = useSettings((s) => s.density)
  const reduceMotion = useSettings((s) => s.reduceMotion)
  const colourTheme = useSettings((s) => s.theme)
  const [osReduce, setOsReduce] = React.useState(() =>
    typeof matchMedia === 'function'
      ? matchMedia('(prefers-reduced-motion: reduce)').matches
      : false,
  )
  const [osLight, setOsLight] = React.useState(() =>
    typeof matchMedia === 'function' ? matchMedia('(prefers-color-scheme: light)').matches : false,
  )
  React.useEffect(() => {
    if (typeof matchMedia !== 'function') {
      return
    }
    const mq = matchMedia('(prefers-reduced-motion: reduce)')
    const onChange = () => setOsReduce(mq.matches)
    mq.addEventListener('change', onChange)
    return () => mq.removeEventListener('change', onChange)
  }, [])
  React.useEffect(() => {
    if (typeof matchMedia !== 'function') {
      return
    }
    const mq = matchMedia('(prefers-color-scheme: light)')
    const onChange = () => setOsLight(mq.matches)
    mq.addEventListener('change', onChange)
    return () => mq.removeEventListener('change', onChange)
  }, [])
  const theme = React.useMemo(
    () =>
      createMortarTheme(isAccent(accent) ? accent : defaultAccent, {
        compact: density === 'compact',
        reduceMotion: honourReducedMotion(reduceMotion || 'system', osReduce),
        mode: honourTheme(colourTheme || 'dark', osLight),
      }),
    [accent, colourTheme, density, osLight, osReduce, reduceMotion],
  )
  return <ThemeProvider theme={theme}>{children}</ThemeProvider>
}
