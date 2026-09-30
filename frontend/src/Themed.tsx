import { ThemeProvider } from '@mui/material'
import React from 'react'
import { useSettings } from './settings/store.ts'
import { buildTheme, isAccent } from './settings/theme.ts'
import { defaultAccent } from './theme/accents.ts'

export function Themed({ children }: { children: React.ReactNode }) {
  const accent = useSettings((s) => s.accent)
  const translucent = useSettings((s) => s.translucent)
  const theme = React.useMemo(
    () => buildTheme(isAccent(accent) ? accent : defaultAccent, translucent),
    [accent, translucent],
  )
  return <ThemeProvider theme={theme}>{children}</ThemeProvider>
}
