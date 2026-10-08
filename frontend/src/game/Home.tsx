import { useLingui } from '@lingui/react/macro'
import { Box, Button, List, ListItem, Typography } from '@mui/material'
import { List as ListIcon, Settings2, Share2, TriangleAlert } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { modsLabel, problemsLabel, updatesLabel } from '../i18n/counts.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { When } from '../i18n/When.tsx'
import { useBadges } from '../mods/badges.ts'
import { problemCount } from '../mods/lookup.ts'
import { useMods } from '../mods/store.ts'
import { useUpdates } from '../mods/updates.ts'
import { useNav } from '../nav/store.ts'
import { CrashHintCard } from '../profiles/CrashHintCard.tsx'
import { userModCount } from '../profiles/count.ts'
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
import { glance, renameSurface, savesView } from './homeView.ts'
import { LaunchMenu } from './LaunchMenu.tsx'
import { ProfileMenu } from './ProfileMenu.tsx'
import { useRenameRequest } from './renameRequest.ts'
import { useTab } from './tab.ts'
import { useLastRun } from './useLastRun.ts'

const SEP = ' · '

function AtAGlance({ profile, game }: { profile: Profile; game: string }) {
  const { t } = useLingui()
  const setTab = useTab((s) => s.setTab)
  const setReviewing = useUpdates((s) => s.setReviewing)
  const badges = useBadges((s) => s.byProfile[profile.id])
  const lastRun = useLastRun(game, profile.id, String(profile.updated))
  const when = formatWhen(lastRun)
  const problems = useMods((s) => (s.problems === null ? null : problemCount(s.problems)))
  const view = glance(userModCount(profile), badges?.updates ?? 0, problems)
  return (
    <HomePanel title={t`At a glance`} card="glance">
      <Typography>
        {modsLabel(view.mods)}
        {view.updates > 0 ? (
          <>
            {SEP}
            <Button
              size="small"
              color="primary"
              sx={{ minWidth: 0, p: 0, fontSize: 'inherit', verticalAlign: 'baseline' }}
              onClick={() => {
                setTab('mods')
                setReviewing(true)
              }}
            >
              {updatesLabel(view.updates)}
            </Button>
          </>
        ) : null}
      </Typography>
      {view.problems === null ? null : (
        <Button
          size="small"
          color={view.problems > 0 ? 'warning' : 'inherit'}
          startIcon={view.problems > 0 ? <TriangleAlert size={14} /> : undefined}
          sx={{ p: 0, minWidth: 0, fontSize: 'inherit' }}
          onClick={() => setTab('problems')}
        >
          {view.problems > 0 ? problemsLabel(view.problems) : t`No problems`}
        </Button>
      )}
      <Typography>{lastRun ? t`Last played ${when}` : t`Not played yet`}</Typography>
      <Typography variant="body2" color="text.secondary">
        {t`Created`} <When value={profile.created} />
        {SEP}
        {t`Updated`} <When value={profile.updated} />
      </Typography>
    </HomePanel>
  )
}

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
          sx={{ minHeight: space.row, gap: space.gap, justifyContent: 'space-between' }}
        >
          <Typography noWrap={true} sx={{ minWidth: 0 }}>
            {saveName(save)}
          </Typography>
          <Typography
            variant="body2"
            color="text.secondary"
            sx={{ flexShrink: 0, whiteSpace: 'nowrap' }}
          >
            {calendar ? t`Year ${calendar.year} ${seasons[calendar.season] ?? ''}` : '—'}
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
          alignItems: 'start',
          px: space.gutter,
          py: space.gutter,
          // Saves right after At a glance means no Changes card shares its row, so it takes the whole row.
          '& [data-card="glance"] + [data-card="saves"]': { gridColumn: 'span 3' },
          [compact]: {
            gridTemplateColumns: 'minmax(0, 1fr)',
            '& [data-card="glance"] + [data-card="saves"]': { gridColumn: 'auto' },
          },
        }}
      >
        <CrashHintCard game={game} profileId={profile.id} crashedAt={crashedAt} />
        <HomeNotes profile={profile} />
        <AtAGlance profile={profile} game={game} />
        <SinceLastRun game={game} profileId={profile.id} />
        <SavesPanel />
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
