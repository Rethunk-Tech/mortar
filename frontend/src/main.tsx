import '@wailsio/runtime'
import { I18nProvider } from '@lingui/react'
import { CssBaseline, ThemeProvider } from '@mui/material'
import React from 'react'
import ReactDOM from 'react-dom/client'
import { App } from './App.tsx'
import { i18n } from './i18n/index.ts'
import { initLaunch } from './launch/store.ts'
import { initSettings, useSettings } from './settings/store.ts'
import { buildTheme, isAccent } from './settings/theme.ts'
import { defaultAccent } from './theme/accents.ts'
import './theme/fonts.ts'

function Themed({ children }: { children: React.ReactNode }) {
  const accent = useSettings((s) => s.accent)
  const translucent = useSettings((s) => s.translucent)
  const theme = React.useMemo(
    () => buildTheme(isAccent(accent) ? accent : defaultAccent, translucent),
    [accent, translucent],
  )
  return <ThemeProvider theme={theme}>{children}</ThemeProvider>
}

initSettings().catch(console.error)
initLaunch()

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <I18nProvider i18n={i18n}>
      <Themed>
        <CssBaseline />
        <App />
      </Themed>
    </I18nProvider>
  </React.StrictMode>,
)
