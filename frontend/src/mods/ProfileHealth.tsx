import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Chip, Tooltip } from '@mui/material'
import { useState } from 'react'
import { Runs } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { HealthHistory } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { problemsLabel, updatesLabel } from '../i18n/counts.ts'
import { useSettings } from '../settings/store.ts'
import { healthView } from './badgeDisplay.ts'
import type { Counts } from './badges.ts'
import { HealthTooltipContent } from './HealthTooltipContent.tsx'

export function ProfileHealth({
  counts,
  game,
  profileId,
}: {
  counts?: Counts | undefined
  game?: string
  profileId?: string
}) {
  const { t } = useLingui()
  const mode = useSettings((s) => s.sidebarBadges)
  const view = healthView(counts, mode, {
    missing: (n) =>
      t`${plural(n, { one: '# missing requirement', other: '# missing requirements' })}`,
    problems: (n) => problemsLabel(n),
    updates: (n) => updatesLabel(n),
  })
  const [detail, setDetail] = useState<{
    history: NonNullable<Awaited<ReturnType<typeof HealthHistory>>>
    runs: NonNullable<Awaited<ReturnType<typeof Runs>>>
  } | null>(null)

  const loadDetail = () => {
    if (!(game && profileId) || detail !== null) {
      return
    }
    Promise.all([HealthHistory(game, profileId), Runs(game, profileId)])
      .then(([history, runs]) => {
        setDetail({ history: history ?? [], runs: runs ?? [] })
      })
      .catch(() => {
        setDetail({ history: [], runs: [] })
      })
  }

  if (view.tone === 'none') {
    return null
  }

  const tooltipTitle =
    game && profileId && detail !== null ? (
      <HealthTooltipContent summary={view.tooltip} history={detail.history} runs={detail.runs} />
    ) : (
      view.tooltip
    )

  return (
    <Tooltip title={tooltipTitle} disableInteractive={true} onOpen={loadDetail}>
      <Chip
        size="small"
        // No tab stop of its own: it sits inside a profile card, which is a button, and its label joins that
        // button's name.
        role="img"
        aria-label={view.tooltip}
        color={view.tone === 'red' ? 'error' : 'warning'}
        label={view.value}
        sx={{ flexShrink: 0, ml: 0.5, fontWeight: 700 }}
      />
    </Tooltip>
  )
}
