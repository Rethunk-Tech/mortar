import { useLingui } from '@lingui/react/macro'
import { useEffect, useState } from 'react'
import { Get } from '../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { MainScreen } from './game/MainScreen.tsx'
import { GameSelect } from './games/GameSelect.tsx'
import { loadGameStatus } from './games/status.ts'
import { useNav } from './nav/store.ts'
import { ProfilesPage } from './profiles/ProfilesPage.tsx'
import { SettingsPage } from './settings/SettingsPage.tsx'
import { AppFrame } from './shell/AppFrame.tsx'
import { useToasts } from './toasts/store.ts'
import { ToastHost } from './toasts/ToastHost.tsx'

export function App() {
  const { t } = useLingui()
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
          .push({ kind: 'error', title: t`Could not read your games`, body: String(e) }),
      )
      .finally(() => setReady(true))
  }, [t])
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.key === ',') {
        e.preventDefault()
        useNav.getState().openSettings()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])
  return (
    <>
      <AppFrame>
        {route.name === 'settings' ? <SettingsPage section={route.section} /> : null}
        {route.name === 'profiles' ? <ProfilesPage /> : null}
        {route.name === 'game' ? <MainScreen game={route.game} /> : null}
        {ready && route.name === 'game-select' ? <GameSelect /> : null}
      </AppFrame>
      <ToastHost />
    </>
  )
}
