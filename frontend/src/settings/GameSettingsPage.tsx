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
import { List } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import { useCurrentGame } from '../nav/currentGame.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { GraphicsApiRow } from './GraphicsApiRow.tsx'
import { PrefKeys } from './PrefRow.tsx'
import { SettingsSection } from './SettingsSection.tsx'
import { SettingsShell, type ShellPage } from './SettingsShell.tsx'
import { HistoryUsageRows } from './sections/DataHistory.tsx'
import { DismissedFolders } from './sections/DismissedFolders.tsx'
import { BackupsPage, ExtraModsFolder, GameFolder, LoaderPage } from './sections/GameSettings.tsx'
import { SourceOrder } from './sections/SourceOrder.tsx'
import { StreamOverlay } from './sections/StreamOverlay.tsx'
import { useSettings } from './store.ts'
import { usePerProfileRelated } from './usePerProfileRelated.tsx'

type GamePage = 'install' | 'loader' | 'play' | 'mods' | 'backups' | 'console' | 'streaming'

function useGameInstall(game: string) {
  const [folder, setFolder] = useState('')
  const [store, setStore] = useState('')
  const [installs, setInstalls] = useState<Install[]>([])
  const [version, setVersion] = useState('')
  const load = useCallback(() => {
    // The install list alone; Steam's status is not needed here and its failure must not blank the folder.
    List()
      .then((games) => {
        const g = (games ?? []).find((x) => x.id === game)
        setFolder(g?.installDir ?? '')
        setStore(g?.store ?? '')
        setInstalls(g?.installs ?? [])
      })
      .catch(reportUnexpected)
  }, [game])
  useEffect(load, [load])
  return { folder, store, installs, version, setVersion, load }
}

function GamePages({
  page,
  setPage,
  query,
}: {
  page: GamePage
  setPage: (p: GamePage) => void
  query: string
}) {
  const { t } = useLingui()
  const name = useProfiles((s) => s.game?.name ?? '')
  const close = useNav((s) => s.closeGameSettings)
  const game = useCurrentGame()
  const g = useGameInstall(game)
  const loader = useProfiles((s) => s.game?.loaders?.[0])
  const { related, dialog } = usePerProfileRelated(game)
  const pages: ShellPage<GamePage>[] = [
    { id: 'install', label: t`Install`, icon: FolderOpen },
    ...(loader
      ? [{ id: 'loader' as const, label: loader.name, icon: Puzzle, groupEnd: true }]
      : []),
    { id: 'play', label: t`Play`, icon: Play },
    { id: 'mods', label: t`Mods`, icon: Package },
    { id: 'backups', label: t`Save backups`, icon: Archive },
    ...(loader?.console
      ? [{ id: 'console' as const, label: t`Console`, icon: SquareTerminal }]
      : []),
    ...(loader?.overlay
      ? [{ id: 'streaming' as const, label: t`Streaming`, icon: RadioIcon }]
      : []),
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
      case 'loader':
        return loader ? (
          <LoaderPage key={g.folder} loader={loader} onVersion={g.setVersion} />
        ) : null
      case 'play':
        return (
          <SettingsSection title={t`Play`}>
            <PrefKeys
              keys={[
                'defaultLaunchMethod',
                ...(loader?.id === 'smapi' ? ['showSmapiConsole'] : []),
                ...(loader?.introSkip ? ['skipIntro'] : []),
                'updateModsBeforePlayDefault',
              ]}
              game={game}
            />
            <GraphicsApiRow game={game} />
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
                  // Only SMAPI skips dot-named folders; other loaders load what is inside them.
                  ...(loader?.id === 'smapi' ? ['showDotHiddenMods'] : []),
                ]}
                game={game}
              />
              <ExtraModsFolder />
              <SourceOrder />
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
        return loader?.overlay ? <StreamOverlay game={game} /> : null
    }
  }
  return (
    <>
      <SettingsShell
        title={t`${name} settings`}
        backLabel={t`Back to ${name}`}
        onBack={close}
        pages={pages}
        current={page}
        onPage={setPage}
        render={render}
        initialQuery={query}
        related={related}
      />
      {dialog}
    </>
  )
}

// A new folder or store choice re-reads the install from scratch.
export function GameSettingsPage() {
  const game = useCurrentGame()
  const override = useSettings((s) => s.gameFolders?.[game] ?? '')
  const chosen = useSettings((s) => s.gameStores?.[game] ?? '')
  const query = useNav((s) => (s.route.name === 'game-settings' ? (s.route.query ?? '') : ''))
  const [page, setPage] = useState<GamePage>('install')
  return <GamePages key={`${override}:${chosen}`} page={page} setPage={setPage} query={query} />
}
