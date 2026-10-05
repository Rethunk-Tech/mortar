import { useLingui } from '@lingui/react/macro'
import { Box, Chip, Divider, Tab, Tabs, type TabsActions } from '@mui/material'
import { Settings2, Share2 } from 'lucide-react'
import { lazy, Suspense, useEffect, useRef } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { BrowseHost } from '../browse/BrowseHost.tsx'
import { ModsTab } from '../mods/ModsTab.tsx'
import { ProblemActions, ProblemsTab } from '../mods/ProblemsTab.tsx'
import { useNav } from '../nav/store.ts'
import { SinceLastRun } from '../profiles/SinceLastRun.tsx'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { openShare } from '../share/store.ts'
import { usePasteLink } from '../share/usePasteLink.ts'
import { ErrorBoundary } from '../shell/ErrorBoundary.tsx'
import { IconAction } from '../shell/IconAction.tsx'
import { ToolsMenu } from '../tools/ToolsMenu.tsx'
import { Hero } from './Hero.tsx'
import { type TabId, useTab } from './tab.ts'

// Tabs other than Mods, Problems and Browse load on first open, which keeps them out of the startup bundle.
const ConsoleTab = lazy(() =>
  import('../console/ConsoleTab.tsx').then((m) => ({ default: m.ConsoleTab })),
)
const LoadOrderTab = lazy(() =>
  import('../mods/LoadOrderTab.tsx').then((m) => ({ default: m.LoadOrderTab })),
)
const NotesTab = lazy(() => import('../notes/NotesTab.tsx').then((m) => ({ default: m.NotesTab })))
const SavesTab = lazy(() => import('../saves/SavesTab.tsx').then((m) => ({ default: m.SavesTab })))
const PerformanceTab = lazy(() =>
  import('../console/PerformanceTab.tsx').then((m) => ({ default: m.PerformanceTab })),
)

// The open profile's loader, whose capabilities decide which tabs exist.
function useProfileLoader() {
  const game = useProfiles((s) => s.game)
  const loaderId = useProfiles((s) => openProfileOf(s)?.loader) || game?.loaderId
  return game?.loaders?.find((l) => l.id === loaderId)
}

function WorkspaceTabs({ problemsTabCount }: { problemsTabCount: number | null }) {
  const { t } = useLingui()
  const loader = useProfileLoader()
  const tab = useTab((s) => s.tab)
  const setTab = useTab((s) => s.setTab)
  const hidden =
    (tab === 'load-order' && !loader?.order) ||
    (tab === 'console' && !loader?.console) ||
    (tab === 'performance' && !loader?.startup)
  // A tab saved from another game or loader is not there; fall back to Mods.
  useEffect(() => {
    if (loader && hidden) {
      setTab('mods')
    }
  }, [loader, hidden, setTab])
  const actions = useRef<TabsActions>(null)
  const root = useRef<HTMLDivElement>(null)
  // MUI places the underline when the value changes, but a tab can widen afterwards (the Problems count arrives
  // later, the bold selected label lands, a font loads), moving the tabs after it; re-place it on every resize.
  useEffect(() => {
    const el = root.current
    if (!el) {
      return
    }
    const observer = new ResizeObserver(() => actions.current?.updateIndicator())
    for (const tabEl of el.querySelectorAll('[role="tab"]')) {
      observer.observe(tabEl)
    }
    return () => observer.disconnect()
  }, [])
  return (
    <Tabs
      action={actions}
      ref={root}
      value={tab}
      onChange={(_, value: TabId) => setTab(value)}
      aria-label={t`Profile sections`}
      sx={{
        minHeight: 44,
        '& .MuiTabs-indicator': { height: 2 },
        '& .MuiTab-root': {
          minHeight: 44,
          minWidth: 0,
          px: '14px',
          fontSize: 14,
          fontWeight: 400,
          color: 'text.secondary',
          '&.Mui-selected': { color: 'var(--mortar-ink)', fontWeight: 600 },
        },
      }}
    >
      <Tab value="browse" label={t`Browse`} data-tour="browse-tab" />
      <Tab value="mods" label={t`Mods`} data-tour="mods-tab" />
      <Tab
        value="problems"
        data-tour="problems-tab"
        label={
          <Box component="span" sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.75 }}>
            {t`Problems`}
            {problemsTabCount !== null && problemsTabCount > 0 ? (
              <Chip
                size="small"
                label={problemsTabCount}
                sx={{ height: 20, fontSize: 12, fontWeight: 600 }}
              />
            ) : null}
          </Box>
        }
      />
      {loader?.order ? <Tab value="load-order" label={t`Load order`} /> : null}
      <Tab value="saves" label={t`Saves`} />
      <Tab value="notes" label={t`Notes`} />
      {loader?.console ? <Tab value="console" label={t`Console`} /> : null}
      {loader?.startup ? <Tab value="performance" label={t`Performance`} /> : null}
    </Tabs>
  )
}

function WorkspaceActions({
  profile,
  game,
  gameName,
}: {
  profile: Profile
  game: string
  gameName: string
}) {
  const { t } = useLingui()
  const tab = useTab((s) => s.tab)
  const openGameSettings = useNav((s) => s.openGameSettings)
  return (
    <Box sx={{ display: 'flex', gap: 0.75 }}>
      {tab === 'problems' ? (
        <>
          <ProblemActions />
          <Divider orientation="vertical" flexItem={true} sx={{ mx: 0.5 }} />
        </>
      ) : null}
      <ToolsMenu game={game} profileID={profile.id} />
      <IconAction
        label={t`Share`}
        icon={<Share2 size={16} />}
        onClick={() => openShare(profile.id)}
      />
      <IconAction
        label={t`${gameName} settings`}
        icon={<Settings2 size={16} />}
        onClick={openGameSettings}
      />
    </Box>
  )
}

export function ProfileWorkspace({
  profile,
  game,
  gameName,
  problemsTabCount,
}: {
  profile: Profile
  game: string
  gameName: string
  problemsTabCount: number | null
}) {
  const tab = useTab((s) => s.tab)
  usePasteLink(profile.id)
  return (
    <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
      <Hero key={`hero-${profile.id}`} profile={profile} game={game} />
      <SinceLastRun game={game} profileId={profile.id} />
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          px: 2,
          borderBottom: '1px solid var(--mortar-hairline)',
          flexShrink: 0,
        }}
      >
        <WorkspaceTabs problemsTabCount={problemsTabCount} />
        <Box sx={{ flexGrow: 1 }} />
        <WorkspaceActions profile={profile} game={game} gameName={gameName} />
      </Box>
      <ErrorBoundary resetKey={`${tab}-${profile.id}`}>
        <Suspense fallback={null}>
          {tab === 'console' ? <ConsoleTab game={game} /> : null}
          {tab === 'performance' ? <PerformanceTab game={game} /> : null}
          {tab === 'notes' ? <NotesTab key={`notes-${profile.id}`} profile={profile} /> : null}
          {tab === 'saves' ? <SavesTab profile={profile} game={game} /> : null}
          {tab === 'load-order' ? (
            <LoadOrderTab key={`load-order-${profile.id}`} profile={profile} game={game} />
          ) : null}
        </Suspense>
        {tab === 'browse' ? (
          <BrowseHost key={`browse-${profile.id}`} game={game} profileID={profile.id} />
        ) : null}
        {tab === 'mods' ? <ModsTab key={`mods-${profile.id}`} profile={profile} /> : null}
        {tab === 'problems' ? (
          <Box
            key={`problems-${profile.id}`}
            sx={{
              flex: 1,
              minHeight: 0,
              overflowY: 'auto',
              display: 'flex',
              flexDirection: 'column',
            }}
          >
            <ProblemsTab />
          </Box>
        ) : null}
      </ErrorBoundary>
    </Box>
  )
}
