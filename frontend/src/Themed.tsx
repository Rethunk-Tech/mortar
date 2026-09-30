import { ThemeProvider } from '@mui/material'
import React from 'react'
import { useSettings } from './settings/store.ts'
import { buildTheme, isAccent } from './settings/theme.ts'
import { defaultAccent } from './theme/accents.ts'

export function Themed({ children }: { children: React.ReactNode }) {
  const accent = useSettings((s) => s.accent)
  const solid = useSettings((s) => s.background === 'solid')
  const theme = React.useMemo(
    () => buildTheme(isAccent(accent) ? accent : defaultAccent, solid),
    [accent, solid],
  )
  return <ThemeProvider theme={theme}>{children}</ThemeProvider>
}
