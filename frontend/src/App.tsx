import { useEffect, useState } from 'react'
import { Get } from '../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { GameSelect } from './games/GameSelect.tsx'
import { loadGameStatus } from './games/status.ts'
import { useNav } from './nav/store.ts'
import { SettingsDialog } from './settings/SettingsDialog.tsx'
import { AppFrame } from './shell/AppFrame.tsx'
import { useToasts } from './toasts/store.ts'
import { ToastHost } from './toasts/ToastHost.tsx'

export function App() {
  const route = useNav((s) => s.route)
  const [ready, setReady] = useState(false)
  useEffect(() => {
    Promise.all([Get(), loadGameStatus()])
      .then(([settings, { games }]) => {
        const last = games.find((g) => g.id === settings.lastGame)
        if (last?.id === 'stardew' && last.available && last.installed) {
          useNav.getState().openGame(last.id)
        }
      })
      .catch((e: unknown) =>
        useToasts
          .getState()
          .push({ kind: 'error', title: 'Could not read your games', body: String(e) }),
      )
      .finally(() => setReady(true))
  }, [])
  return (
    <>
      <AppFrame>{ready && route.name === 'game-select' ? <GameSelect /> : null}</AppFrame>
      <ToastHost />
      <SettingsDialog />
    </>
  )
}
