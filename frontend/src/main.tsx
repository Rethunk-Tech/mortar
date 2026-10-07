import '@wailsio/runtime'
import { I18nProvider } from '@lingui/react'
import { CssBaseline } from '@mui/material'
import React from 'react'
import ReactDOM from 'react-dom/client'
import { Get } from '../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { App } from './App.tsx'
import { logRenderError } from './diag/renderErrors.ts'
import { initNewDownloads } from './downloads/newDownloads.ts'
import { activateLanguage, i18n } from './i18n/index.ts'
import { initInstallAsks } from './install/store.ts'
import { IncomingPrompt } from './lan/IncomingPrompt.tsx'
import { initIncoming } from './lan/incoming.ts'
import { OutgoingProgress } from './lan/OutgoingProgress.tsx'
import { initLaunch } from './launch/events.ts'
import { initPlayMode } from './launch/playMode.ts'
import { initPlayRequests } from './launch/playRequests.ts'
import { initLoader } from './loader/store.ts'
import { initNexusSeen } from './mods/nexusDetails.ts'
import { initUpdateDigestToast } from './mods/updateDigestToast.ts'
import { initNxm } from './nxm/store.ts'
import { initProfilesChanged } from './profiles/store.ts'
import { initQueue } from './queue/store.ts'
import { initQuit } from './quit.ts'
import { initScheduledBackups } from './saves/scheduledBackups.ts'
import { initNexus } from './settings/nexus.ts'
import { initSettings } from './settings/store.ts'
import { initShare } from './share/arrivals.ts'
import { initGamepad } from './shell/gamepad.ts'
import { initSync } from './sync/store.ts'
import { Themed } from './Themed.tsx'
import { initTidyReport } from './tidy/report.ts'
import { reportUnexpected } from './toasts/report.ts'
import { initTrayNoticeClick } from './tray/noticeClick.ts'
import { initMortarUpdateBackground } from './updates/background.ts'
import './theme/fonts.ts'

document.addEventListener('contextmenu', (event) => {
  const { target } = event
  const editable =
    target instanceof Element &&
    target.closest('input, textarea, [contenteditable="true"]') !== null
  const selection = globalThis.getSelection()?.toString()
  if (!(editable || selection)) {
    event.preventDefault()
  }
})

await activateLanguage((await Get()).language)
// Settings land before the first render, so the first frame already uses the saved sort, grouping and columns.
await initSettings().catch(reportUnexpected)
initInstallAsks()
initMortarUpdateBackground()
initUpdateDigestToast()
initNewDownloads()
initNexus().catch(reportUnexpected)
initNexusSeen().catch(reportUnexpected)
initLaunch()
initSync()
initTrayNoticeClick()
initLoader()
initNxm().catch(reportUnexpected)
initQueue().catch(reportUnexpected)
initProfilesChanged()
// Play mode is known before the first render, so the full window never draws behind the play prompt.
await initPlayMode().catch(reportUnexpected)
initPlayRequests()
initShare().catch(reportUnexpected)
initIncoming().catch(reportUnexpected)
initQuit()
initScheduledBackups()
initTidyReport()
initGamepad()

ReactDOM.createRoot(document.getElementById('root') as HTMLElement, {
  onUncaughtError: logRenderError,
  onCaughtError: logRenderError,
  onRecoverableError: logRenderError,
}).render(
  <React.StrictMode>
    <I18nProvider i18n={i18n}>
      <Themed>
        <CssBaseline />
        <App />
        <IncomingPrompt />
        <OutgoingProgress />
      </Themed>
    </I18nProvider>
  </React.StrictMode>,
)
