import { useLingui } from '@lingui/react/macro'
import { Box, Chip, Divider, Tab, Tabs } from '@mui/material'
import { Settings2, Share2 } from 'lucide-react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { BrowseHost } from '../browse/BrowseHost.tsx'
import { ConsoleTab } from '../console/ConsoleTab.tsx'
import { PerformancePanel } from '../console/PerformancePanel.tsx'
import { LoadOrderTab } from '../mods/LoadOrderTab.tsx'
import { ModsTab } from '../mods/ModsTab.tsx'
import { ProblemActions, ProblemsTab } from '../mods/ProblemsTab.tsx'
import { useNav } from '../nav/store.ts'
import { NotesTab } from '../notes/NotesTab.tsx'
import { SinceLastRun } from '../profiles/SinceLastRun.tsx'
import { SavesTab } from '../saves/SavesTab.tsx'
import { openShare } from '../share/store.ts'
import { usePasteLink } from '../share/usePasteLink.ts'
import { ErrorBoundary } from '../shell/ErrorBoundary.tsx'
import { IconAction } from '../shell/IconAction.tsx'
import { ToolsMenu } from '../tools/ToolsMenu.tsx'
import { Hero } from './Hero.tsx'
import { type TabId, useTab } from './tab.ts'

function WorkspaceTabs({ problemsTabCount }: { problemsTabCount: number | null }) {
  const { t } = useLingui()
  const tab = useTab((s) => s.tab)
  const setTab = useTab((s) => s.setTab)
  return (
    <Tabs
      value={tab}
      onChange={(_, value: TabId) => setTab(value)}
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
      <Tab value="browse" label={t`Browse`} />
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
      <Tab value="load-order" label={t`Load order`} />
      <Tab value="saves" label={t`Saves`} />
      <Tab value="notes" label={t`Notes`} />
      <Tab value="console" label={t`Console`} />
      <Tab value="performance" label={t`Performance`} />
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
        {tab === 'console' ? <ConsoleTab game={game} /> : null}
        {tab === 'performance' ? <PerformancePanel game={game} /> : null}
        {tab === 'notes' ? <NotesTab key={`notes-${profile.id}`} profile={profile} /> : null}
        {tab === 'saves' ? <SavesTab profile={profile} game={game} /> : null}
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
        {tab === 'load-order' ? (
          <LoadOrderTab key={`load-order-${profile.id}`} profile={profile} game={game} />
        ) : null}
      </ErrorBoundary>
    </Box>
  )
}
