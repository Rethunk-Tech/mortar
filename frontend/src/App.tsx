import { useLingui } from '@lingui/react/macro'
import { useEffect, useState } from 'react'
import { Get } from '../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { CommandPalette } from './commandPalette/CommandPalette.tsx'
import { FirstRun } from './firstrun/FirstRun.tsx'
import { GameSetup } from './firstrun/GameSetup.tsx'
import { gameSetupNeeded } from './firstrun/needed.ts'
import { FirstRunTour } from './firstrunTour/FirstRunTour.tsx'
import { FomodDialog } from './fomod/Dialog.tsx'
import { MainScreen } from './game/MainScreen.tsx'
import { GameSelect } from './games/GameSelect.tsx'
import { loadGameStatus } from './games/status.ts'
import { LaunchLayer } from './launch/LaunchLayer.tsx'
import { overlayGame, useLaunch } from './launch/store.ts'
import { PinReasonDialog } from './mods/pinReasonDialog.tsx'
import { isGameId, useNav } from './nav/store.ts'
import { ArrivalDialog } from './nxm/ArrivalDialog.tsx'
import { ProfilesPage } from './profiles/ProfilesPage.tsx'
import { QuitPrompt } from './QuitPrompt.tsx'
import { GameSettingsPage } from './settings/GameSettingsPage.tsx'
import { SettingsPage } from './settings/SettingsPage.tsx'
import { useStartupChecks } from './settings/startupChecks.ts'
import { useAppShortcuts } from './settings/useShortcuts.ts'
import { ImportDialog } from './share/ImportDialog.tsx'
import { ShareDialog } from './share/ShareDialog.tsx'
import { AppFrame } from './shell/AppFrame.tsx'
import { BugReportDialog } from './shell/BugReportDialog.tsx'
import { ErrorBoundary } from './shell/ErrorBoundary.tsx'
import { toastError } from './toasts/report.ts'
import { ToastHost } from './toasts/ToastHost.tsx'
import { UpdateReadyBanner } from './updates/UpdateReadyBanner.tsx'
import { WhatsNewDialog } from './updates/WhatsNewDialog.tsx'

export function App() {
  useAppShortcuts()
  useStartupChecks()
  const { t } = useLingui()
  const route = useNav((s) => s.route)
  const game = overlayGame(
    route.name,
    route.name === 'game' ? route.game : '',
    useLaunch((s) => s.status?.game ?? ''),
  )
  const [ready, setReady] = useState(false)
  useEffect(() => {
    Promise.all([Get(), loadGameStatus()])
      .then(async ([settings, { games }]) => {
        if (!settings.launchersConfirmed) {
          useNav.getState().openSetup()
          return
        }
        if (settings.startScreen === 'gameselect') {
          return
        }
        const last = games.find((g) => g.id === settings.lastGame)
        if (last && isGameId(last.id) && last.available && !(await gameSetupNeeded(last))) {
          useNav.getState().openGame(last.id)
        }
      })
      .catch((e: unknown) => toastError(t`Could not read your games`, e))
      .finally(() => setReady(true))
  }, [t])
  return (
    <>
      <AppFrame>
        <UpdateReadyBanner />
        <ErrorBoundary resetKey={route.name}>
          {route.name === 'settings' ? <SettingsPage section={route.section} /> : null}
          {route.name === 'game-settings' ? <GameSettingsPage /> : null}
          {route.name === 'profiles' ? <ProfilesPage /> : null}
          {route.name === 'game' ? <MainScreen game={route.game} /> : null}
          {route.name === 'setup' ? <FirstRun /> : null}
          {route.name === 'game-setup' ? <GameSetup game={route.game} /> : null}
          {ready && route.name === 'game-select' ? <GameSelect /> : null}
        </ErrorBoundary>
      </AppFrame>
      <LaunchLayer game={game} />
      <ArrivalDialog />
      <FomodDialog />
      <CommandPalette />
      <FirstRunTour />
      <PinReasonDialog />
      <ShareDialog />
      <ImportDialog />
      <WhatsNewDialog />
      <BugReportDialog />
      <QuitPrompt />
      <ToastHost />
    </>
  )
}
