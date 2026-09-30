import '@wailsio/runtime'
import { I18nProvider } from '@lingui/react'
import { CssBaseline } from '@mui/material'
import React from 'react'
import ReactDOM from 'react-dom/client'
import { App } from './App.tsx'
import { i18n } from './i18n/index.ts'
import { initLaunch } from './launch/store.ts'
import { initSettings } from './settings/store.ts'
import { Themed } from './Themed.tsx'
import { reportUnexpected } from './toasts/report.ts'
import './theme/fonts.ts'

initSettings().catch(reportUnexpected)
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
