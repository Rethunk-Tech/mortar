import '@wailsio/runtime'
import { I18nProvider } from '@lingui/react'
import { CssBaseline, ThemeProvider } from '@mui/material'
import React from 'react'
import ReactDOM from 'react-dom/client'
import { App } from './App.tsx'
import { i18n } from './i18n/index.ts'
import { defaultAccent } from './theme/accents.ts'
import './theme/fonts.ts'
import { createMortarTheme } from './theme/theme.ts'

const theme = createMortarTheme(defaultAccent)

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <I18nProvider i18n={i18n}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <App />
      </ThemeProvider>
    </I18nProvider>
  </React.StrictMode>,
)
