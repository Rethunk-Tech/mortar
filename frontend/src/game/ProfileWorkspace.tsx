import { Box } from '@mui/material'
import type { ReactNode } from 'react'
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
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { SavesTab } from '../saves/SavesTab.tsx'
import { ScreenshotsTab } from '../screenshots/ScreenshotsTab.tsx'
import { usePasteLink } from '../share/usePasteLink.ts'
import { ErrorBoundary } from '../shell/ErrorBoundary.tsx'
import { Home } from './Home.tsx'
import { TabHeader } from './TabHeader.tsx'
import { PANEL_ID, tabDomId, useTab } from './tab.ts'

// The tab that shows the whole open profile reads it here, so a save re-renders that tab and not the workspace.
function WithProfile({ show }: { show: (profile: Profile) => ReactNode }) {
  const profile = useProfiles(openProfileOf)
  return profile ? show(profile) : null
}

// Rereads the config list on each save of the open profile; its own component, so a save re-renders only this.
function ConfigListFollower({ game, id }: { game: string; id: string }) {
  const updated = useProfiles((s) => openProfileOf(s)?.updated ?? '')
  useConfigListLoaded(game, { id, updated })
  return null
}

export function ProfileWorkspace({ game, gameName }: { game: string; gameName: string }) {
  const tab = useTab((s) => s.tab)
  const id = useProfiles((s) => s.openId)
  const name = useProfiles((s) => openProfileOf(s)?.name ?? '')
  usePasteLink(id)
  return (
    <Box
      role="tabpanel"
      id={PANEL_ID}
      aria-labelledby={tabDomId(tab)}
      sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}
    >
      <ConfigListFollower game={game} id={id} />
      {tab === 'home' ? (
        <WithProfile
          key={`home-${id}`}
          show={(profile) => <Home profile={profile} game={game} gameName={gameName} />}
        />
      ) : null}
      {tab === 'home' ? null : <TabHeader />}
      <ErrorBoundary resetKey={`${tab}-${id}`}>
        {tab === 'console' ? <ConsoleTab game={game} /> : null}
        {tab === 'performance' ? <PerformanceTab game={game} /> : null}
        {tab === 'config' ? (
          <WithProfile show={(profile) => <ConfigTab profile={profile} game={game} />} />
        ) : null}
        {tab === 'saves' ? (
          <WithProfile show={(profile) => <SavesTab profile={profile} game={game} />} />
        ) : null}
        {tab === 'screenshots' ? <ScreenshotsTab game={game} /> : null}
        {tab === 'load-order' ? (
          <WithProfile
            key={`load-order-${id}`}
            show={(profile) => <LoadOrderTab profile={profile} game={game} />}
          />
        ) : null}
        {tab === 'browse' ? <BrowseHost key={`browse-${id}`} game={game} profileID={id} /> : null}
        {tab === 'mods' ? (
          <WithProfile key={`mods-${id}`} show={(profile) => <ModsTab profile={profile} />} />
        ) : null}
        {tab === 'problems' ? (
          <Box
            key={`problems-${id}`}
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
        <DuplicateDialog profileName={name} />
      </ErrorBoundary>
    </Box>
  )
}
