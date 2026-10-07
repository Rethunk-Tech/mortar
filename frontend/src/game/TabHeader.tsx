import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import { alpha } from '@mui/material/styles'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { modsLabel, problemsLabel, updatesLabel } from '../i18n/counts.ts'
import { useBadges } from '../mods/badges.ts'
import { problemCount } from '../mods/lookup.ts'
import { useMods } from '../mods/store.ts'
import { useUpdates } from '../mods/updates.ts'
import { colorHex } from '../profiles/appearance.ts'
import { userModCount } from '../profiles/count.ts'
import { useSaves } from '../saves/store.ts'
import { space } from '../theme/density.ts'
import { compact, saveFits } from './compact.ts'
import { usePageActionsSlot } from './pageActions.ts'
import { type TabId, useTab } from './tab.ts'
import { notesFirstLine, pageActionsSx } from './tabHeader.ts'

const TINT = 0.16

function Chip({
  label,
  tone,
  title,
  onClick,
}: {
  label: string
  tone?: 'primary.main' | 'warning.main'
  title?: string
  onClick: () => void
}) {
  return (
    <ButtonBase
      onClick={onClick}
      title={title}
      sx={{
        height: 26,
        px: 1.25,
        flexShrink: 0,
        whiteSpace: 'nowrap',
        borderRadius: '13px',
        border: '1px solid',
        borderColor: tone ?? 'var(--mortar-hairline-16)',
        bgcolor: 'var(--mortar-hairline-faint)',
        color: tone ?? 'text.primary',
        fontSize: 12,
      }}
    >
      {label}
    </ButtonBase>
  )
}

// The strip every tab but Home opens with: the profile's name and status, and a slot for the tab's own buttons.
export function TabHeader({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const setTab = useTab((s) => s.setTab)
  const setReviewing = useUpdates((s) => s.setReviewing)
  const badges = useBadges((s) => s.byProfile[profile.id])
  const fits = useSaves((s) => s.fits)
  const setSlot = usePageActionsSlot((s) => s.setSlot)
  const updates = badges?.updates ?? 0
  const problems = useMods((s) => (s.problems === null ? 0 : problemCount(s.problems)))
  const saves = saveFits(fits)
  const notes = notesFirstLine(profile.notes ?? '')
  const go = (tab: TabId) => () => setTab(tab)
  // A save that records no mod list cannot fit or miss, so a game with only those shows the plain count.
  const savesLabel = (s: typeof saves) =>
    s.recorded > 0
      ? t`${s.fitting} of ${s.recorded} saves fit`
      : plural(s.total, { one: '# save', other: '# saves' })
  return (
    <Box
      component="header"
      sx={{
        height: `calc(${space.control} + 2 * ${space.gap})`,
        flexShrink: 0,
        display: 'flex',
        alignItems: 'center',
        gap: space.gap,
        px: space.gutter,
        bgcolor: (th) => alpha(colorHex(profile.color) ?? th.palette.primary.main, TINT),
        borderBottom: '1px solid var(--mortar-hairline)',
      }}
    >
      <Typography noWrap={true} component="h1" sx={{ fontSize: 18, fontWeight: 700, minWidth: 0 }}>
        {profile.name}
      </Typography>
      <Box sx={{ display: 'flex', gap: space.gap, flexShrink: 0 }}>
        <Chip label={modsLabel(userModCount(profile))} onClick={go('mods')} />
        {updates > 0 ? (
          <Chip
            label={updatesLabel(updates)}
            tone="primary.main"
            onClick={() => {
              setTab('mods')
              setReviewing(true)
            }}
          />
        ) : null}
        {problems > 0 ? (
          <Chip label={problemsLabel(problems)} tone="warning.main" onClick={go('problems')} />
        ) : null}
        {saves.total > 0 ? (
          <Chip
            label={savesLabel(saves)}
            title={t`Saves whose recorded mod list this profile covers, out of the saves that record one`}
            onClick={go('saves')}
          />
        ) : null}
      </Box>
      <Typography
        noWrap={true}
        title={notes}
        sx={{
          minWidth: 0,
          fontSize: 13,
          color: 'text.secondary',
          [compact]: { display: 'none' },
        }}
      >
        {notes}
      </Typography>
      <Box sx={{ flex: '1 0 12px' }} />
      <Box
        ref={setSlot}
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: space.gap,
          flexShrink: 0,
          ...pageActionsSx,
        }}
      />
    </Box>
  )
}
