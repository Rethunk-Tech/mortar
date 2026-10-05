import { useLingui } from '@lingui/react/macro'
import {
  Archive,
  FolderOpen,
  Package,
  Play,
  Puzzle,
  Radio as RadioIcon,
  SquareTerminal,
} from 'lucide-react'
import { type ReactNode, useCallback, useEffect, useState } from 'react'
import type { Install } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { loadGameStatus } from '../games/status.ts'
import { useCurrentGame } from '../nav/currentGame.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { PrefKeys } from './PrefRow.tsx'
import { SettingsSection } from './SettingsSection.tsx'
import { SettingsShell, type ShellPage } from './SettingsShell.tsx'
import { HistoryUsageRows } from './sections/DataHistory.tsx'
import { DismissedFolders } from './sections/DismissedFolders.tsx'
import { BackupsPage, ExtraModsFolder, GameFolder, SmapiPage } from './sections/GameSettings.tsx'
import { StreamOverlay } from './sections/StreamOverlay.tsx'
import { useSettings } from './store.ts'

type GamePage = 'install' | 'smapi' | 'play' | 'mods' | 'backups' | 'console' | 'streaming'

function useGameInstall(game: string) {
  const [folder, setFolder] = useState('')
  const [store, setStore] = useState('')
  const [installs, setInstalls] = useState<Install[]>([])
  const [version, setVersion] = useState('')
  const load = useCallback(() => {
    loadGameStatus()
      .then((s) => {
        const g = s.games.find((x) => x.id === game)
        setFolder(g?.installDir ?? '')
        setStore(g?.store ?? '')
        setInstalls(g?.installs ?? [])
      })
      .catch(reportUnexpected)
  }, [game])
  useEffect(load, [load])
  return { folder, store, installs, version, setVersion, load }
}

function GamePages({ page, setPage }: { page: GamePage; setPage: (p: GamePage) => void }) {
  const { t } = useLingui()
  const name = useProfiles((s) => s.game?.name ?? '')
  const close = useNav((s) => s.closeGameSettings)
  const game = useCurrentGame()
  const g = useGameInstall(game)
  const pages: ShellPage<GamePage>[] = [
    { id: 'install', label: t`Install`, icon: FolderOpen },
    { id: 'smapi', label: t`SMAPI`, icon: Puzzle, groupEnd: true },
    { id: 'play', label: t`Play`, icon: Play },
    { id: 'mods', label: t`Mods`, icon: Package },
    { id: 'backups', label: t`Save backups`, icon: Archive },
    { id: 'console', label: t`Console`, icon: SquareTerminal },
    { id: 'streaming', label: t`Streaming`, icon: RadioIcon },
  ]
  const render = (id: GamePage): ReactNode => {
    switch (id) {
      case 'install':
        return (
          <GameFolder
            folder={g.folder}
            store={g.store}
            installs={g.installs}
            onRefresh={g.load}
            versionNote={g.version ? t` · ${name} ${g.version}` : ''}
          />
        )
      case 'smapi':
        return <SmapiPage key={g.folder} onVersion={g.setVersion} />
      case 'play':
        return (
          <SettingsSection title={t`Play`}>
            <PrefKeys
              keys={['defaultLaunchMethod', 'showSmapiConsole', 'updateModsBeforePlayDefault']}
              game={game}
            />
          </SettingsSection>
        )
      case 'mods':
        return (
          <>
            <SettingsSection title={t`Mods`}>
              <PrefKeys
                keys={[
                  'enableRequirements',
                  'missingRequirements',
                  'cosmeticConflicts',
                  'conflictScanDepth',
                  'offerNewDownloads',
                  'oldFilesOnUpdate',
                  'showDotHiddenMods',
                ]}
                game={game}
              />
              <ExtraModsFolder />
            </SettingsSection>
            <DismissedFolders />
            <SettingsSection title={t`${name} profile history`}>
              <HistoryUsageRows />
            </SettingsSection>
          </>
        )
      case 'backups':
        return <BackupsPage />
      case 'console':
        return (
          <SettingsSection title={t`Console`}>
            <PrefKeys
              keys={[
                'runsKept',
                'consoleLogCap',
                'consoleLevel',
                'consoleTimestamps',
                'consoleFollow',
              ]}
              game={game}
            />
          </SettingsSection>
        )
      default:
        return <StreamOverlay />
    }
  }
  return (
    <SettingsShell
      title={t`${name} settings`}
      backLabel={t`Back to ${name}`}
      onBack={close}
      pages={pages}
      current={page}
      onPage={setPage}
      render={render}
    />
  )
}

// A new folder or store choice re-reads the install from scratch.
export function GameSettingsPage() {
  const game = useCurrentGame()
  const override = useSettings((s) => s.gameFolders?.[game] ?? '')
  const chosen = useSettings((s) => s.gameStores?.[game] ?? '')
  const [page, setPage] = useState<GamePage>('install')
  return <GamePages key={`${override}:${chosen}`} page={page} setPage={setPage} />
}
