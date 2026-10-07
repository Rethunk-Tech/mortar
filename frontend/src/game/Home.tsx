import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { Settings2, Share2, TriangleAlert } from 'lucide-react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { modsLabel, problemsLabel, updatesLabel } from '../i18n/counts.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useBadges } from '../mods/badges.ts'
import { useUpdates } from '../mods/updates.ts'
import { useNav } from '../nav/store.ts'
import { CrashHintCard } from '../profiles/CrashHintCard.tsx'
import { userModCount } from '../profiles/count.ts'
import { SinceLastRun } from '../profiles/SinceLastRun.tsx'
import { filterAndSortSaves } from '../saves/filterAndSortSaves.ts'
import { saveName } from '../saves/saveName.ts'
import { useSaves } from '../saves/store.ts'
import { openShare } from '../share/store.ts'
import { compact } from './compact.ts'
import { Hero } from './Hero.tsx'
import { HomeButton } from './HomeButton.tsx'
import { HomeNotes } from './HomeNotes.tsx'
import { HomePanel } from './HomePanel.tsx'
import { glance, savesView } from './homeView.ts'
import { LaunchMenu } from './LaunchMenu.tsx'
import { ProfileMenu } from './ProfileMenu.tsx'
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
  const view = glance(userModCount(profile), badges?.updates ?? 0, badges?.problems ?? null)
  return (
    <HomePanel title={t`At a glance`}>
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
    </HomePanel>
  )
}

function SavesPanel() {
  const { t } = useLingui()
  const setTab = useTab((s) => s.setTab)
  const fits = useSaves((s) => s.fits)
  const { shown, more } = savesView(filterAndSortSaves(fits, ''))
  const seasons = [t`Spring`, t`Summer`, t`Fall`, t`Winter`]
  return (
    <HomePanel title={t`Saves`}>
      {shown.length === 0 ? (
        <Typography color="text.secondary">{t`No saves yet`}</Typography>
      ) : (
        shown.map(({ save, calendar }) => {
          const name = saveName(save)
          const line = calendar
            ? t`${name} · Year ${calendar.year} ${seasons[calendar.season] ?? ''}`
            : name
          return (
            <Typography key={save.folder} noWrap={true} sx={{ maxWidth: '100%' }}>
              {line}
            </Typography>
          )
        })
      )}
      <Button size="small" onClick={() => setTab('saves')}>
        {more > 0 ? t`${more} more · All saves` : t`All saves`}
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
          gap: 1.25,
          px: 4,
          pt: 2,
          [compact]: { px: 2 },
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
          gap: 2,
          px: 4,
          py: 2.5,
          [compact]: { gridTemplateColumns: 'minmax(0, 1fr)', px: 2 },
        }}
      >
        <CrashHintCard game={game} profileId={profile.id} crashedAt={crashedAt} />
        <HomeNotes profile={profile} />
        <AtAGlance profile={profile} game={game} />
        <SinceLastRun game={game} profileId={profile.id} />
        <SavesPanel />
      </Box>
    </Box>
  )
}
