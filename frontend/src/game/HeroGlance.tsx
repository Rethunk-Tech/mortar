import { useLingui } from '@lingui/react/macro'
import { Box, Link } from '@mui/material'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { modsLabel, problemsLabel, updatesLabel } from '../i18n/counts.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { When } from '../i18n/When.tsx'
import { useBadges } from '../mods/badges.ts'
import { problemCount } from '../mods/lookup.ts'
import { useMods } from '../mods/store.ts'
import { useUpdates } from '../mods/updates.ts'
import { userModCount } from '../profiles/count.ts'
import { ON_ART_SHADOW, ON_ART_TEXT } from './onArt.ts'
import { useTab } from './tab.ts'
import { useLastRun } from './useLastRun.ts'

const FONT_PX = 13

// The profile's facts on the hero, under its name: mods, updates, problems, last played, created and updated.
export function HeroGlance({
  profile,
  game,
  compactAt,
}: {
  profile: Profile
  game: string
  compactAt: string
}) {
  const { t } = useLingui()
  const setTab = useTab((s) => s.setTab)
  const setReviewing = useUpdates((s) => s.setReviewing)
  const updates = useBadges((s) => s.byProfile[profile.id]?.updates ?? 0)
  const problems = useMods((s) => (s.problems === null ? null : problemCount(s.problems)))
  const lastRun = useLastRun(game, profile.id, String(profile.updated))
  const when = formatWhen(lastRun)
  const parts = [
    <span key="mods">{modsLabel(userModCount(profile))}</span>,
    updates > 0 ? (
      <Link
        key="updates"
        component="button"
        underline="hover"
        color="inherit"
        onClick={() => {
          setTab('mods')
          setReviewing(true)
        }}
      >
        {updatesLabel(updates)}
      </Link>
    ) : null,
    problems === null ? null : (
      <Link
        key="problems"
        component="button"
        underline="hover"
        color="inherit"
        sx={problems > 0 ? { color: 'warning.light', fontWeight: 600 } : undefined}
        onClick={() => setTab('problems')}
      >
        {problems > 0 ? problemsLabel(problems) : t`No problems`}
      </Link>
    ),
    <span key="played">{lastRun ? t`Last played ${when}` : t`Not played yet`}</span>,
    <span key="created">
      {t`Created`} <When value={profile.created} />
    </span>,
    <span key="updated">
      {t`Updated`} <When value={profile.updated} />
    </span>,
  ].filter((p) => p !== null)
  return (
    <Box
      sx={{
        mt: 0.5,
        display: 'flex',
        flexWrap: 'wrap',
        columnGap: 0.75,
        fontSize: FONT_PX,
        color: ON_ART_TEXT,
        textShadow: ON_ART_SHADOW,
        [compactAt]: { display: 'none' },
        '& button': { fontSize: 'inherit', p: 0, verticalAlign: 'baseline' },
      }}
    >
      {parts.map((p, i) => (
        <Box component="span" key={p.key} sx={{ whiteSpace: 'nowrap' }}>
          {i === 0 ? null : '· '}
          {p}
        </Box>
      ))}
    </Box>
  )
}
