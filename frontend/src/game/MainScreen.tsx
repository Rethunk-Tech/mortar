import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { useEffect } from 'react'
import { HelpDialog } from '../console/HelpDialog.tsx'
import { useGameInfo } from '../games/info.ts'
import { LoaderBanner } from '../loader/Banner.tsx'
import { useProfiles } from '../profiles/store.ts'
import { space } from '../theme/density.ts'
import { Detail } from './Detail.tsx'
import { Sidebar } from './Sidebar.tsx'
import { useSidebarBadges } from './useSidebarProfiles.ts'

// Its own component: the badge loader follows every profile and mod change, which must not re-render the screen.
function SidebarBadges({ game }: { game: string }) {
  useSidebarBadges(game)
  return null
}

// The live catalog has this game's entry only for a newer Mortar, so this build shows its own, older one.
function NewerMortarBanner({ game }: { game: string }) {
  const { t } = useLingui()
  const info = useGameInfo(game)
  if (!info?.newerMortar) {
    return null
  }
  return (
    <Box
      role="status"
      sx={{
        px: space.pad,
        py: space.gap,
        flexShrink: 0,
        bgcolor: 'var(--mortar-panel)',
        borderLeft: '4px solid',
        borderLeftColor: 'info.main',
      }}
    >
      <Typography sx={{ fontSize: 'medium.fontSize' }}>
        {t`A newer Mortar has updates for ${{ name: info.name }}. Update Mortar to get them.`}
      </Typography>
    </Box>
  )
}

export function MainScreen({ game }: { game: string }) {
  const load = useProfiles((s) => s.load)
  useEffect(() => {
    load(game)
  }, [game, load])
  return (
    <Box sx={{ position: 'relative', height: '100%', display: 'flex', flexDirection: 'column' }}>
      <SidebarBadges game={game} />
      <NewerMortarBanner game={game} />
      <LoaderBanner game={game} />
      <Box sx={{ flex: 1, minHeight: 0, display: 'flex' }}>
        <Sidebar game={game} />
        <Detail />
      </Box>
      <HelpDialog game={game} />
    </Box>
  )
}
