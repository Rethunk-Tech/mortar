import { ThemeProvider } from '@mui/material'
import React from 'react'
import { useSettings } from './settings/store.ts'
import { isAccent } from './settings/theme.ts'
import { defaultAccent } from './theme/accents.ts'
import { createMortarTheme } from './theme/theme.ts'

export function Themed({ children }: { children: React.ReactNode }) {
  const accent = useSettings((s) => s.accent)
  const theme = React.useMemo(
    () => createMortarTheme(isAccent(accent) ? accent : defaultAccent),
    [accent],
  )
  return <ThemeProvider theme={theme}>{children}</ThemeProvider>
}
