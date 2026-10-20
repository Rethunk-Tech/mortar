import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import { alpha } from '@mui/material/styles'
import { memo } from 'react'
import { modsLabel, problemsLabel, updatesLabel } from '../i18n/counts.ts'
import { useBadges } from '../mods/badges.ts'
import { problemCount } from '../mods/lookup.ts'
import { useMods } from '../mods/store.ts'
import { useUpdates } from '../mods/updates.ts'
import { colorHex } from '../profiles/appearance.ts'
import { userModCount } from '../profiles/count.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
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
        px: space.gap,
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
// It reads the open profile's few fields itself, so a change elsewhere in the profile (a mod switched) leaves it alone.
export const TabHeader = memo(function TabHeaderView() {
  const { t } = useLingui()
  const id = useProfiles((s) => s.openId)
  const name = useProfiles((s) => openProfileOf(s)?.name ?? '')
  const color = useProfiles((s) => openProfileOf(s)?.color)
  const notes = useProfiles((s) => notesFirstLine(openProfileOf(s)?.notes ?? ''))
  const modCount = useProfiles((s) => {
    const open = openProfileOf(s)
    return open ? userModCount(open) : 0
  })
  const setTab = useTab((s) => s.setTab)
  const setReviewing = useUpdates((s) => s.setReviewing)
  const badges = useBadges((s) => s.byProfile[id])
  const fits = useSaves((s) => s.fits)
  const setSlot = usePageActionsSlot((s) => s.setSlot)
  const updates = badges?.updates ?? 0
  const problems = useMods((s) => (s.problems === null ? 0 : problemCount(s.problems)))
  const saves = saveFits(fits)
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
        bgcolor: (th) => alpha(colorHex(color) ?? th.palette.primary.main, TINT),
        borderBottom: '1px solid var(--mortar-hairline)',
      }}
    >
      <Typography noWrap={true} component="h1" sx={{ fontSize: 18, fontWeight: 700, minWidth: 0 }}>
        {name}
      </Typography>
      <Box sx={{ display: 'flex', gap: space.gap, flexShrink: 0 }}>
        <Chip label={modsLabel(modCount)} onClick={go('mods')} />
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
})
