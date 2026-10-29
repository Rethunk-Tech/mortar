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
import { type ReactNode, useCallback, useEffect, useReducer, useState } from 'react'
import type { Install } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { List } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import { useGameInfo } from '../games/info.ts'
import { useCurrentGame } from '../nav/currentGame.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { GraphicsApiRow } from './GraphicsApiRow.tsx'
import { gamePrefs } from './gamePrefs.ts'
import { PrefKeys } from './PrefRow.tsx'
import { ProtonRow } from './ProtonRow.tsx'
import { ScopeSwitch } from './ScopeSwitch.tsx'
import { SettingsSection } from './SettingsSection.tsx'
import { SettingsShell, type ShellPage } from './SettingsShell.tsx'
import { SharedOnProfile } from './SharedOnProfile.tsx'
import { ALVRProgram } from './sections/ALVRProgram.tsx'
import { ConsoleLevels } from './sections/ConsoleLevels.tsx'
import { HistoryUsageRows } from './sections/DataHistory.tsx'
import { DismissedFolders } from './sections/DismissedFolders.tsx'
import { ExtraModsFolders } from './sections/ExtraModsFolders.tsx'
import { BackupsPage, GameFolder, LoaderPage } from './sections/GameSettings.tsx'
import { GameSync } from './sections/GameSync.tsx'
import { HiddenLogSources } from './sections/HiddenLogSources.tsx'
import { HiddenMods } from './sections/HiddenMods.tsx'
import { LoaderProfileSettings, ProfileGameOptions } from './sections/ProfileScopeSections.tsx'
import { SourceOrder } from './sections/SourceOrder.tsx'
import { StreamOverlay } from './sections/StreamOverlay.tsx'
import { useSettingsScope } from './settingsScope.ts'
import { useSettings } from './store.ts'
import { usePerProfileRelated } from './usePerProfileRelated.tsx'
import { useProfileScope } from './useProfileScope.ts'
import { setSettingsQuery } from './useSettingsSearch.ts'

type GamePage = 'install' | 'loader' | 'play' | 'mods' | 'backups' | 'console' | 'streaming'

function modsPrefKeys(
  info: { hasOptions: boolean; hasCaches: boolean; sources: string[] | null } | undefined,
  loader: { id: string; builtin: boolean; assets: boolean } | undefined,
): string[] {
  const loaderId = loader?.id
  return [
    // Requirements come from mod manifests, which a game with no loader of its own has none of.
    ...(loader?.builtin
      ? []
      : ['enableRequirements', 'disableDependencies', 'missingRequirements']),
    // Both concern content packs' asset edits, which only a loader whose framework indexes them has.
    ...(loader?.assets ? ['cosmeticConflicts', 'conflictScanDepth'] : []),
    'offerNewDownloads',
    'oldFilesOnUpdate',
    ...(info?.sources?.includes('thunderstore') ? ['dependencyVersions'] : []),
    ...(info?.hasOptions ? ['gameSettingsMode'] : []),
    ...(info?.hasCaches ? ['cacheClearing'] : []),
    // Only SMAPI skips dot-named folders; other loaders load what is inside them.
    ...(loaderId === 'smapi' ? ['showDotHiddenMods'] : []),
  ]
}

function useGameInstall(game: string) {
  const [folder, setFolder] = useState('')
  const [store, setStore] = useState('')
  const [installs, setInstalls] = useState<Install[]>([])
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
  return { folder, store, installs, load }
}

// The install folder loads here, not in GamePages, so its arrival redraws this page and not every page beside it.
function InstallPage({
  game,
  versionNote,
  onFolderChanged,
}: {
  game: string
  versionNote: string
  onFolderChanged: () => void
}) {
  const g = useGameInstall(game)
  return (
    <GameFolder
      folder={g.folder}
      store={g.store}
      installs={g.installs}
      onRefresh={() => {
        g.load()
        onFolderChanged()
      }}
      versionNote={versionNote}
    />
  )
}

// Whether the game starts directly: its own choice, an install that has no launcher, or a profile's override.
function useLaunchesDirect(game: string, directInstall: boolean) {
  const gameDirect = useSettings((s) => gamePrefs(s, game).defaultLaunchMethod === 'direct')
  const profileDirect = useProfiles((s) =>
    s.game?.id === game
      ? s.profiles.some((p) => p.overrides?.defaultLaunchMethod === 'direct')
      : false,
  )
  return gameDirect || profileDirect || directInstall
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
  const [version, setVersion] = useState('')
  // The Loader page reads the loader's state again once the player points the game at another folder.
  const [folderChanges, folderChanged] = useReducer((n: number) => n + 1, 0)
  const info = useGameInfo(game)
  const loader = useProfiles((s) => s.game?.loaders?.[0])
  const launchDirect = useLaunchesDirect(game, info?.directInstall === true)
  // The ALVR program row shows wherever ALVR can start first: the game's choice or a profile's override.
  const gameALVR = useSettings((s) => gamePrefs(s, game).startFirst === 'alvr')
  const profileALVR = useProfiles((s) =>
    s.game?.id === game ? s.profiles.some((p) => p.overrides?.startFirst === 'alvr') : false,
  )
  const startsALVR = gameALVR || profileALVR
  const related = usePerProfileRelated(game, () => {
    setSettingsQuery('')
    useSettingsScope.setState({ profile: true })
    setPage('play')
  })
  // On the open profile's scope the page also shows what is only ever per profile.
  const profileScope = useProfileScope()
  const openId = useProfiles((s) => s.openId ?? '')
  const ownProfile = profileScope && openId !== '' ? openId : ''
  const pages: ShellPage<GamePage>[] = [
    { id: 'install', label: t`Install`, icon: FolderOpen },
    ...(loader && !loader.builtin
      ? [{ id: 'loader' as const, label: loader.name, icon: Puzzle, groupEnd: true }]
      : []),
    { id: 'play', label: t`Play`, icon: Play },
    { id: 'mods', label: t`Mods`, icon: Package },
    ...(info?.hasSaves ? [{ id: 'backups' as const, label: t`Save backups`, icon: Archive }] : []),
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
          <SharedOnProfile>
            <InstallPage
              game={game}
              versionNote={version ? `${name} ${version}` : ''}
              onFolderChanged={folderChanged}
            />
          </SharedOnProfile>
        )
      case 'loader':
        return loader && !loader.builtin ? (
          <>
            <SharedOnProfile>
              <LoaderPage key={folderChanges} loader={loader} onVersion={setVersion} />
            </SharedOnProfile>
            <LoaderProfileSettings game={game} profileId={ownProfile} loaderName={loader.name} />
          </>
        ) : null
      case 'play':
        return (
          <SettingsSection title={t`Play`}>
            <PrefKeys
              keys={[
                'defaultLaunchMethod',
                'startFirst',
                ...(launchDirect ? ['instances', 'delaySeconds'] : []),
                ...(loader?.id === 'smapi' ? ['showSmapiConsole'] : []),
                ...(loader?.introSkip ? ['skipIntro'] : []),
                'updateModsBeforePlayDefault',
                'skipPlayCheck',
                'pauseDownloadsWhilePlaying',
              ]}
              game={game}
            />
            {startsALVR ? (
              <SharedOnProfile>
                <ALVRProgram />
              </SharedOnProfile>
            ) : null}
            <GraphicsApiRow game={game} />
            <ProtonRow game={game} />
            <ProfileGameOptions game={game} profileId={ownProfile} />
          </SettingsSection>
        )
      case 'mods':
        return (
          <>
            <SettingsSection title={t`Mods`}>
              <PrefKeys keys={modsPrefKeys(info, loader)} game={game} />
              <SharedOnProfile>
                <ExtraModsFolders />
                <GameSync />
                <SourceOrder />
                <HiddenMods />
              </SharedOnProfile>
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
          <>
            <SettingsSection title={t`Console`}>
              <PrefKeys keys={['runsKept', 'consoleLogCap']} game={game} />
              <ConsoleLevels game={game} />
              <PrefKeys keys={['consoleTimestamps', 'consoleFollow']} game={game} />
            </SettingsSection>
            <SharedOnProfile>
              <HiddenLogSources />
            </SharedOnProfile>
          </>
        )
      default:
        return loader?.overlay ? (
          <SharedOnProfile>
            <StreamOverlay game={game} />
          </SharedOnProfile>
        ) : null
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
      actions={Object.fromEntries(
        pages.map((p) => [p.id, <ScopeSwitch key="scope" gameName={name} />]),
      )}
      initialQuery={query}
      {...(related ? { related } : {})}
    />
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
