import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { Plus } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { ProfilesWithMod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import type {
  Fit,
  Lack,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/models.ts'
import { useLocked } from '../mods/useLocked.ts'
import { useProfiles } from '../profiles/store.ts'
import { download, type Want } from '../queue/actions.ts'
import { useQueue } from '../queue/store.ts'
import { pendingFor } from '../queue/totals.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { LackChipActions } from './lackChipActions.tsx'
import { wantFor } from './wantFor.ts'

const nowrap = { whiteSpace: 'nowrap' } as const

function LackChip({
  fit,
  lack,
  profile,
  game,
}: {
  fit: Fit
  lack: Lack
  profile: Profile
  game: string
}) {
  const { t } = useLingui()
  const { name } = lack
  const locked = useLocked()
  const [source, setSource] = useState<{ id: string; name: string } | null>(null)
  useEffect(() => {
    ProfilesWithMod(game, lack.uniqueId)
      .then((rows) => {
        const row = (rows ?? []).find((candidate) => candidate.profileId !== profile.id)
        const sourceName = useProfiles
          .getState()
          .profiles.find((candidate) => candidate.id === row?.profileId)?.name
        setSource(row && sourceName ? { id: row.profileId, name: sourceName } : null)
      })
      .catch(reportUnexpected)
  }, [game, lack.uniqueId, profile.id])
  const want = lack.disabled ? null : wantFor(lack)
  const queued = useQueue((s) =>
    want ? pendingFor(s.state.items, profile.id, want.modId ?? 0, want.repo ?? '') : false,
  )
  let plusTitle = t`Add to this profile`
  if (queued) {
    plusTitle = t`Queued`
  }
  if (locked) {
    plusTitle = t`Stop the game to change mods.`
  }
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 0.25,
        pl: 1,
        pr: 0.25,
        borderRadius: '4px',
        bgcolor: 'var(--mortar-overlay-30)',
      }}
    >
      <Typography sx={{ fontSize: 13, ...nowrap }}>
        {lack.disabled ? t`${name} (switched off)` : lack.name}
      </Typography>
      <LackChipActions
        fit={fit}
        lack={lack}
        profile={profile}
        game={game}
        locked={locked}
        queued={queued}
        plusTitle={plusTitle}
        source={source}
      />
    </Box>
  )
}

export function AddAll({ missing, profile }: { missing: Lack[]; profile: Profile }) {
  const { t } = useLingui()
  const [pending, run] = usePending()
  const locked = useLocked()
  const items = useQueue((s) => s.state.items)
  const wants = missing
    .map((lack) => (lack.disabled ? null : wantFor(lack)))
    .filter((want): want is Want => want !== null)
    .filter((want) => !pendingFor(items, profile.id, want.modId ?? 0, want.repo ?? ''))
  if (wants.length < 2) {
    return null
  }
  return (
    <DisabledReason title={t`Stop the game to change mods.`} disabled={locked}>
      <Button
        size="small"
        variant="outlined"
        color="inherit"
        startIcon={<Plus size={14} />}
        disabled={pending || locked}
        onClick={() => run(() => download(wants))}
        sx={nowrap}
      >
        {t`Add all ${wants.length}`}
      </Button>
    </DisabledReason>
  )
}

export function MissingChips({ fit, profile, game }: { fit: Fit; profile: Profile; game: string }) {
  const missing = fit.missing ?? []
  if (missing.length === 0) {
    return null
  }
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, flexWrap: 'wrap' }}>
      {missing.map((lack) => (
        <LackChip key={lack.uniqueId} fit={fit} lack={lack} profile={profile} game={game} />
      ))}
    </Box>
  )
}
