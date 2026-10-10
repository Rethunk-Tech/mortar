import { useLingui } from '@lingui/react/macro'
import { lazy, Suspense, useCallback, useEffect, useState } from 'react'
import { Get } from '../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { CommandPalette } from './commandPalette/CommandPalette.tsx'
import { gameSetupNeeded } from './firstrun/needed.ts'
import { FirstRunTour } from './firstrunTour/FirstRunTour.tsx'
import { FomodDialog } from './fomod/Dialog.tsx'
import { MainScreen } from './game/MainScreen.tsx'
import { GameSelect } from './games/GameSelect.tsx'
import { loadGameStatus } from './games/status.ts'
import { DownloadsDialog } from './install/DownloadsDialog.tsx'
import { HandoffDialog } from './install/HandoffDialog.tsx'
import { LaunchLayer } from './launch/LaunchLayer.tsx'
import { usePlayMode } from './launch/playModeState.ts'
import { overlayGame, useLaunch } from './launch/store.ts'
import { PinReasonDialog } from './mods/pinReasonDialog.tsx'
import { isGameId, useNav } from './nav/store.ts'
import { ArrivalDialog } from './nxm/ArrivalDialog.tsx'
import { ProfilesPage } from './profiles/ProfilesPage.tsx'
import { QuitPrompt } from './QuitPrompt.tsx'
import { useStartupChecks } from './settings/startupChecks.ts'
import { useAppShortcuts } from './settings/useShortcuts.ts'
import { ImportDialog } from './share/ImportDialog.tsx'
import { ShareDialog } from './share/ShareDialog.tsx'
import { AppFrame } from './shell/AppFrame.tsx'
import { BugReportDialog } from './shell/BugReportDialog.tsx'
import { ErrorBoundary } from './shell/ErrorBoundary.tsx'
import { AvOverrideDialog } from './toasts/AvOverrideDialog.tsx'
import { toastError } from './toasts/report.ts'
import { ToastHost } from './toasts/ToastHost.tsx'
import { UpdateReadyBanner } from './updates/UpdateReadyBanner.tsx'
import { WhatsNewDialog } from './updates/WhatsNewDialog.tsx'

// Pages opened on demand load on first use, which keeps the startup bundle under Vite's chunk size warning.
const FirstRun = lazy(() =>
  import('./firstrun/FirstRun.tsx').then((m) => ({ default: m.FirstRun })),
)
const GameSetup = lazy(() =>
  import('./firstrun/GameSetup.tsx').then((m) => ({ default: m.GameSetup })),
)
const GameSettingsPage = lazy(() =>
  import('./settings/GameSettingsPage.tsx').then((m) => ({ default: m.GameSettingsPage })),
)
const SettingsPage = lazy(() =>
  import('./settings/SettingsPage.tsx').then((m) => ({ default: m.SettingsPage })),
)

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
  const solo = usePlayMode((s) => s.solo)
  const boot = useCallback((): void => {
    // Play mode opens the played profile's game itself, and a later jump to the last game would race it.
    if (usePlayMode.getState().solo) {
      setReady(true)
      return
    }
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
      .catch((e: unknown) => toastError(t`Could not read your games`, e, { retry: boot }))
      .finally(() => setReady(true))
  }, [t])
  useEffect(boot, [boot])
  if (solo) {
    return <LaunchLayer game={game} />
  }
  return (
    <>
      <AppFrame>
        <UpdateReadyBanner />
        <ErrorBoundary resetKey={route.name}>
          <Suspense fallback={null}>
            {route.name === 'settings' ? <SettingsPage section={route.section} /> : null}
            {route.name === 'game-settings' ? <GameSettingsPage /> : null}
            {route.name === 'profiles' ? <ProfilesPage /> : null}
            {route.name === 'game' ? <MainScreen game={route.game} /> : null}
            {route.name === 'setup' ? <FirstRun /> : null}
            {route.name === 'game-setup' ? <GameSetup game={route.game} /> : null}
            {ready && route.name === 'game-select' ? <GameSelect /> : null}
          </Suspense>
        </ErrorBoundary>
      </AppFrame>
      <LaunchLayer game={game} />
      <ArrivalDialog />
      <FomodDialog />
      <CommandPalette />
      <AvOverrideDialog />
      <HandoffDialog />
      <FirstRunTour />
      <PinReasonDialog />
      <ShareDialog />
      <ImportDialog />
      <DownloadsDialog />
      <WhatsNewDialog />
      <BugReportDialog />
      <QuitPrompt />
      <ToastHost />
    </>
  )
}
