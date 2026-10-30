import { i18n } from '@lingui/core'
import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { useLoaded, useStoredState } from '@rethunk/react-load-hooks'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type {
  Fit,
  Gap,
  GapMod,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/models.ts'
import { SaveGap } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { LockedReason } from '../mods/LockedReason.tsx'
import { useLocked } from '../mods/useLocked.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { addRecordedMods } from './recordedActions.ts'

const isText = (v: unknown): v is string => typeof v === 'string'

function stateText(kind: 'missing' | 'disabled' | 'older', m: GapMod): string {
  if (kind === 'disabled') {
    return i18n._(msg`Disabled`).toLowerCase()
  }
  if (kind === 'older') {
    return i18n._(msg`older, profile has ${{ have: m.have }}`)
  }
  return i18n._(msg`Missing`).toLowerCase()
}

export function SaveGapLine({ fit, profile, game }: { fit: Fit; profile: Profile; game: string }) {
  const { t } = useLingui()
  const { data: gap } = useLoaded<Gap | null>(
    () => SaveGap(game, profile.id, fit.folder),
    [game, profile.id, fit.folder],
    null,
    reportUnexpected,
  )
  const [shown, setShown] = useState(false)
  const [pending, run] = usePending()
  const locked = useLocked()
  const missing = gap?.missing ?? []
  const disabled = gap?.disabled ?? []
  const need = missing.length + disabled.length
  // Dismissing remembers which mods were wanted, so the line comes back only when that set changes.
  const wanted = [...missing, ...disabled]
    .map((m) => m.id)
    .sort((a, b) => a.localeCompare(b))
    .join('\n')
  const [dismissed, setDismissed] = useStoredState(
    `mortar.saveGapDismissed.${game}.${profile.id}.${fit.folder}`,
    '',
    isText,
  )
  if (!gap || need === 0 || dismissed === wanted) {
    return null
  }
  const rows = [
    ...missing.map((m) => ({ m, state: stateText('missing', m) })),
    ...disabled.map((m) => ({ m, state: stateText('disabled', m) })),
    ...(gap.versionOlder ?? []).map((m) => ({ m, state: stateText('older', m) })),
  ]
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        <Typography sx={{ flex: 1, fontSize: 'small.fontSize', color: 'warning.main' }}>
          {t`Needs ${plural({ count: need }, { one: '# mod this profile lacks', other: '# mods this profile lacks' })}`}
        </Typography>
        <Button size="small" onClick={() => setShown(!shown)}>
          {shown ? t`Hide` : t`Show`}
        </Button>
        <Button size="small" color="inherit" onClick={() => setDismissed(wanted)}>
          {t`Dismiss`}
        </Button>
        <LockedReason locked={locked}>
          <Button
            size="small"
            variant="outlined"
            color="inherit"
            disabled={pending || locked}
            onClick={() => run(() => addRecordedMods(game, profile.id, fit))}
          >
            {t`Add missing mods`}
          </Button>
        </LockedReason>
      </Box>
      {shown
        ? rows.map(({ m, state }) => (
            <Typography key={`${state}-${m.id}`} sx={{ fontSize: 'small.fontSize' }} noWrap={true}>
              {`${m.name} · ${m.version} · ${state}`}
            </Typography>
          ))
        : null}
    </Box>
  )
}
