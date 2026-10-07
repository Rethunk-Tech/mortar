import { useLingui } from '@lingui/react/macro'
import { Box, Divider } from '@mui/material'
import { Settings2, Share2 } from 'lucide-react'
import { lazy, Suspense } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { BrowseHost } from '../browse/BrowseHost.tsx'
import { ModsTab } from '../mods/ModsTab.tsx'
import { ProblemActions, ProblemsTab } from '../mods/ProblemsTab.tsx'
import { useNav } from '../nav/store.ts'
import { CrashHintCard } from '../profiles/CrashHintCard.tsx'
import { SinceLastRun } from '../profiles/SinceLastRun.tsx'
import { openShare } from '../share/store.ts'
import { usePasteLink } from '../share/usePasteLink.ts'
import { ErrorBoundary } from '../shell/ErrorBoundary.tsx'
import { IconAction } from '../shell/IconAction.tsx'
import { ToolsMenu } from '../tools/ToolsMenu.tsx'
import { Hero } from './Hero.tsx'
import { useTab } from './tab.ts'

// Tabs other than Mods, Problems and Browse load on first open, which keeps them out of the startup bundle.
const ConsoleTab = lazy(() =>
  import('../console/ConsoleTab.tsx').then((m) => ({ default: m.ConsoleTab })),
)
const LoadOrderTab = lazy(() =>
  import('../mods/LoadOrderTab.tsx').then((m) => ({ default: m.LoadOrderTab })),
)
const SavesTab = lazy(() => import('../saves/SavesTab.tsx').then((m) => ({ default: m.SavesTab })))
const PerformanceTab = lazy(() =>
  import('../console/PerformanceTab.tsx').then((m) => ({ default: m.PerformanceTab })),
)

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
        label={t`${{ name: gameName }} settings`}
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
}: {
  profile: Profile
  game: string
  gameName: string
}) {
  const tab = useTab((s) => s.tab)
  usePasteLink(profile.id)
  return (
    <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
      {tab === 'home' ? <Hero key={`hero-${profile.id}`} profile={profile} game={game} /> : null}
      <CrashHintCard game={game} profileId={profile.id} />
      <SinceLastRun game={game} profileId={profile.id} />
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'flex-end',
          minHeight: 44,
          px: 2,
          borderBottom: '1px solid var(--mortar-hairline)',
          flexShrink: 0,
        }}
      >
        <WorkspaceActions profile={profile} game={game} gameName={gameName} />
      </Box>
      <ErrorBoundary resetKey={`${tab}-${profile.id}`}>
        <Suspense fallback={null}>
          {tab === 'console' ? <ConsoleTab game={game} /> : null}
          {tab === 'performance' ? <PerformanceTab game={game} /> : null}
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
