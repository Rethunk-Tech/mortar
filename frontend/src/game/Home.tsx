import { useLingui } from '@lingui/react/macro'
import { Box, Button, List, ListItem, Typography } from '@mui/material'
import { List as ListIcon, Settings2, Share2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useNav } from '../nav/store.ts'
import { CrashHintCard } from '../profiles/CrashHintCard.tsx'
import { EditProfileDialog } from '../profiles/EditProfileDialog.tsx'
import { SinceLastRun } from '../profiles/SinceLastRun.tsx'
import { filterAndSortSaves } from '../saves/filterAndSortSaves.ts'
import { saveName } from '../saves/saveName.ts'
import { useSaves } from '../saves/store.ts'
import { useSettings } from '../settings/store.ts'
import { openShare } from '../share/store.ts'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { space } from '../theme/density.ts'
import { compact } from './compact.ts'
import { Hero } from './Hero.tsx'
import { HomeButton } from './HomeButton.tsx'
import { HomeNotes } from './HomeNotes.tsx'
import { HomePanel } from './HomePanel.tsx'
import { renameSurface, savesView } from './homeView.ts'
import { LaunchMenu } from './LaunchMenu.tsx'
import { ProfileMenu } from './ProfileMenu.tsx'
import { useRenameRequest } from './renameRequest.ts'
import { useTab } from './tab.ts'
import { useLastRun } from './useLastRun.ts'

function SavesList() {
  const { t } = useLingui()
  const fits = useSaves((s) => s.fits)
  const status = useSaves((s) => s.status)
  const error = useSaves((s) => s.error)
  const { shown } = savesView(filterAndSortSaves(fits, ''))
  const seasons = [t`Spring`, t`Summer`, t`Fall`, t`Winter`]
  if (status === 'error') {
    return <Typography color="error">{error}</Typography>
  }
  if (shown.length === 0) {
    return status === 'loading' ? (
      <LoadingRow>{t`Reading your saves…`}</LoadingRow>
    ) : (
      <Typography color="text.secondary">{t`No saves yet`}</Typography>
    )
  }
  return (
    <List disablePadding={true} sx={{ width: '100%' }}>
      {shown.map(({ save, calendar }) => (
        <ListItem
          key={save.folder}
          disablePadding={true}
          sx={{ gap: space.gap, justifyContent: 'space-between' }}
        >
          <Typography noWrap={true} sx={{ minWidth: 0 }}>
            {saveName(save)}
          </Typography>
          <Typography
            variant="body2"
            color="text.secondary"
            sx={{ flexShrink: 0, whiteSpace: 'nowrap' }}
          >
            {calendar ? t`Year ${calendar.year} ${seasons[calendar.season] ?? ''}` : ''}
          </Typography>
        </ListItem>
      ))}
    </List>
  )
}

function SavesPanel() {
  const { t } = useLingui()
  const setTab = useTab((s) => s.setTab)
  const fits = useSaves((s) => s.fits)
  const { total } = savesView(filterAndSortSaves(fits, ''))
  return (
    <HomePanel title={t`Saves`} card="saves">
      <SavesList />
      <Button size="small" startIcon={<ListIcon size={14} />} onClick={() => setTab('saves')}>
        {total > 0 ? t`All saves (${total})` : t`All saves`}
      </Button>
    </HomePanel>
  )
}

export function Home({
  profile,
  game,
  gameName,
}: {
  profile: Profile
  game: string
  gameName: string
}) {
  const { t } = useLingui()
  const openGameSettings = useNav((s) => s.openGameSettings)
  const hero = useSettings((s) => s.profileHero) || 'full'
  const renameId = useRenameRequest((s) => s.id)
  const [renaming, setRenaming] = useState(false)
  useEffect(() => {
    if (renameId === profile.id && renameSurface(hero) === 'dialog') {
      useRenameRequest.getState().clear()
      setRenaming(true)
    }
  }, [renameId, profile.id, hero])
  const crashedAt = useLastRun(game, profile.id, String(profile.updated))
  return (
    <Box
      sx={{ flex: 1, minHeight: 0, overflowY: 'auto', display: 'flex', flexDirection: 'column' }}
    >
      <Hero key={`hero-${profile.id}`} profile={profile} game={game} />
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          flexWrap: 'wrap',
          gap: space.gap,
          px: space.gutter,
          pt: space.gutter,
        }}
      >
        <ProfileMenu profile={profile} />
        <HomeButton icon={<Share2 size={16} />} onClick={() => openShare(profile.id)}>
          {t`Share…`}
        </HomeButton>
        <LaunchMenu game={game} profile={profile} />
        <Box sx={{ flex: '1 0 24px' }} />
        <HomeButton icon={<Settings2 size={16} />} onClick={openGameSettings}>
          {t`${{ name: gameName }} settings`}
        </HomeButton>
      </Box>
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: 'repeat(3, minmax(0, 1fr))',
          gap: space.pad,
          // Cards side by side end level with each other.
          alignItems: 'stretch',
          px: space.gutter,
          py: space.gutter,
          [compact]: {
            gridTemplateColumns: 'minmax(0, 1fr)',
          },
        }}
      >
        <CrashHintCard game={game} profileId={profile.id} crashedAt={crashedAt} />
        <HomeNotes profile={profile} />
        <SavesPanel />
        <SinceLastRun game={game} profileId={profile.id} />
      </Box>
      <EditProfileDialog
        profile={profile}
        open={renaming}
        focusName={true}
        onClose={() => setRenaming(false)}
      />
    </Box>
  )
}
