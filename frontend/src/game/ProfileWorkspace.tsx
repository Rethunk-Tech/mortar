import { Box } from '@mui/material'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { BrowseHost } from '../browse/BrowseHost.tsx'
import { ConsoleTab } from '../console/ConsoleTab.tsx'
import { PerformanceTab } from '../console/PerformanceTab.tsx'
import { DuplicateDialog } from '../mods/DuplicateDialog.tsx'
import { LoadOrderTab } from '../mods/LoadOrderTab.tsx'
import { ModsTab } from '../mods/ModsTab.tsx'
import { ProblemsTab } from '../mods/ProblemsTab.tsx'
import { RemoveDialog } from '../mods/parts.tsx'
import { ConfigTab } from '../mods/typedConfig/ConfigTab.tsx'
import { useConfigListLoaded } from '../mods/typedConfig/configList.ts'
import { SavesTab } from '../saves/SavesTab.tsx'
import { ScreenshotsTab } from '../screenshots/ScreenshotsTab.tsx'
import { usePasteLink } from '../share/usePasteLink.ts'
import { ErrorBoundary } from '../shell/ErrorBoundary.tsx'
import { Home } from './Home.tsx'
import { TabHeader } from './TabHeader.tsx'
import { PANEL_ID, tabDomId, useTab } from './tab.ts'

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
  useConfigListLoaded(game, profile)
  return (
    <Box
      role="tabpanel"
      id={PANEL_ID}
      aria-labelledby={tabDomId(tab)}
      sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}
    >
      {tab === 'home' ? (
        <Home key={`home-${profile.id}`} profile={profile} game={game} gameName={gameName} />
      ) : null}
      {tab === 'home' ? null : <TabHeader />}
      <ErrorBoundary resetKey={`${tab}-${profile.id}`}>
        {tab === 'console' ? <ConsoleTab game={game} /> : null}
        {tab === 'performance' ? <PerformanceTab game={game} /> : null}
        {tab === 'config' ? <ConfigTab profile={profile} game={game} /> : null}
        {tab === 'saves' ? <SavesTab profile={profile} game={game} /> : null}
        {tab === 'screenshots' ? <ScreenshotsTab game={game} /> : null}
        {tab === 'load-order' ? (
          <LoadOrderTab key={`load-order-${profile.id}`} profile={profile} game={game} />
        ) : null}
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
        {/* Mods and Problems both open these, so they live above either tab. */}
        <RemoveDialog />
        <DuplicateDialog profileName={profile.name} />
      </ErrorBoundary>
    </Box>
  )
}
