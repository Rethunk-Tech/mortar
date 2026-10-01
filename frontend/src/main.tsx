import '@wailsio/runtime'
import { I18nProvider } from '@lingui/react'
import { CssBaseline } from '@mui/material'
import React from 'react'
import ReactDOM from 'react-dom/client'
import { App } from './App.tsx'
import { i18n } from './i18n/index.ts'
import { initLaunch } from './launch/store.ts'
import { initLoader } from './loader/store.ts'
import { initNexusSeen } from './mods/nexusDetails.ts'
import { initNxm } from './nxm/store.ts'
import { initQueue } from './queue/store.ts'
import { initNexus } from './settings/nexus.ts'
import { initSettings } from './settings/store.ts'
import { initShare } from './share/arrivals.ts'
import { Themed } from './Themed.tsx'
import { reportUnexpected } from './toasts/report.ts'
import { initTrayNoticeClick } from './tray/noticeClick.ts'
import { initMortarUpdateBackground } from './updates/background.ts'
import './theme/fonts.ts'

initSettings().catch(reportUnexpected)
initMortarUpdateBackground()
initNexus().catch(reportUnexpected)
initNexusSeen().catch(reportUnexpected)
initLaunch()
initTrayNoticeClick()
initLoader()
initNxm().catch(reportUnexpected)
initQueue().catch(reportUnexpected)
initShare().catch(reportUnexpected)

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
