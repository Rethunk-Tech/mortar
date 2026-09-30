import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, IconButton, Tooltip, Typography } from '@mui/material'
import { Browser } from '@wailsio/runtime'
import { ExternalLink, Plus, Power, X } from 'lucide-react'
import type { ReactNode } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import type {
  Fit,
  Lack,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/models.ts'
import { useLocked } from '../mods/useLocked.ts'
import { download, type Want } from '../queue/actions.ts'
import { useQueue } from '../queue/store.ts'
import { pendingFor } from '../queue/totals.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { useSaves } from './store.ts'

const nowrap = { whiteSpace: 'nowrap' } as const

// What the queue needs to add the mod, when Mortar can install it: a Nexus page or a GitHub repository.
function wantFor(lack: Lack): Want | null {
  const { where } = lack
  if (where?.site === 'GitHub' && where.github) {
    return { kind: 'install', repo: where.github, name: where.github }
  }
  if (where?.site === 'Nexus' && where.pageId > 0) {
    return {
      kind: 'install',
      modId: where.pageId,
      fileId: where.fileId,
      name: lack.name,
      fileName: where.fileName,
      version: where.version,
    }
  }
  return null
}

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
  const dismiss = useSaves((s) => s.dismiss)
  const enable = useSaves((s) => s.enable)
  const locked = useLocked()
  const want = lack.disabled ? null : wantFor(lack)
  const queued = useQueue((s) =>
    want ? pendingFor(s.state.items, profile.id, want.modId ?? 0, want.repo ?? '') : false,
  )
  const url = lack.where?.url ?? ''
  const nexusPage = lack.where?.site === 'Nexus'
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 0.25,
        pl: 1,
        pr: 0.25,
        borderRadius: '4px',
        bgcolor: 'rgba(0,0,0,0.3)',
      }}
    >
      <Typography sx={{ fontSize: 13, ...nowrap }}>
        {lack.disabled ? t`${name} (switched off)` : lack.name}
      </Typography>
      {lack.disabled ? (
        <Tooltip title={locked ? t`Stop the game to change mods.` : t`Switch on in this profile`}>
          <span>
            <IconButton
              size="small"
              disabled={locked}
              aria-label={t`Switch on ${name} in this profile`}
              onClick={() => {
                enable(game, profile, lack.uniqueId).catch(reportUnexpected)
              }}
            >
              <Power size={14} />
            </IconButton>
          </span>
        </Tooltip>
      ) : null}
      {want ? (
        <Tooltip title={queued ? t`Queued` : t`Add to this profile`}>
          <span>
            <IconButton
              size="small"
              disabled={queued}
              aria-label={t`Add ${name} to this profile`}
              onClick={() => {
                download([want]).catch(reportUnexpected)
              }}
            >
              <Plus size={14} />
            </IconButton>
          </span>
        </Tooltip>
      ) : null}
      {!(lack.disabled || want) && url ? (
        <Tooltip title={nexusPage ? t`Open on Nexus` : t`Open page`}>
          <IconButton
            size="small"
            aria-label={nexusPage ? t`Open ${name} on Nexus` : t`Open the page of ${name}`}
            onClick={() => {
              Browser.OpenURL(url).catch(reportUnexpected)
            }}
          >
            <ExternalLink size={14} />
          </IconButton>
        </Tooltip>
      ) : null}
      <Tooltip title={t`Dismiss for this save`}>
        <IconButton
          size="small"
          aria-label={t`Dismiss ${name} for this save`}
          onClick={() => {
            dismiss(fit.folder, lack.uniqueId).catch(reportUnexpected)
          }}
        >
          <X size={14} />
        </IconButton>
      </Tooltip>
    </Box>
  )
}

// Queues every mod the save has used that Mortar can install and the queue does not already hold for the profile.
function AddAll({ missing, profile }: { missing: Lack[]; profile: Profile }) {
  const { t } = useLingui()
  const [pending, run] = usePending()
  const items = useQueue((s) => s.state.items)
  const wants = missing
    .map((lack) => (lack.disabled ? null : wantFor(lack)))
    .filter((want): want is Want => want !== null)
    .filter((want) => !pendingFor(items, profile.id, want.modId ?? 0, want.repo ?? ''))
  if (wants.length < 2) {
    return null
  }
  return (
    <Button
      size="small"
      variant="outlined"
      color="inherit"
      startIcon={<Plus size={14} />}
      disabled={pending}
      onClick={() => run(() => download(wants))}
      sx={nowrap}
    >
      {t`Add all ${wants.length}`}
    </Button>
  )
}

function SaveRow({ fit, profile, game }: { fit: Fit; profile: Profile; game: string }) {
  const { t } = useLingui()
  const seasons = [t`Spring`, t`Summer`, t`Fall`, t`Winter`]
  const missing = fit.missing ?? []
  const ok = missing.length === 0
  const played = new Date(fit.played).toLocaleDateString()
  const season = seasons[fit.season] ?? ''
  const { day, year } = fit
  const date = day > 0 ? t`${season} ${day}, year ${year}` : ''
  const meta = [fit.farmer, date, t`last played ${played}`].filter(Boolean).join(' · ')
  const color = ok ? 'success.main' : 'warning.main'
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 1,
        px: 1.75,
        py: 1.5,
        bgcolor: 'rgba(50,50,60,0.78)',
        borderLeft: '4px solid',
        borderLeftColor: color,
        borderRadius: '6px',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
        <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
          <Typography noWrap={true} sx={{ fontSize: 16, fontWeight: 600 }}>
            {fit.farm || fit.folder}
          </Typography>
          <Typography noWrap={true} sx={{ fontSize: 13, color: 'text.secondary' }}>
            {meta}
          </Typography>
        </Box>
        <Box
          sx={{
            px: 1.25,
            py: '3px',
            borderRadius: '12px',
            fontSize: 12,
            fontWeight: 700,
            bgcolor: ok ? 'rgba(12,223,100,0.16)' : 'rgba(243,180,22,0.18)',
            color,
            ...nowrap,
          }}
        >
          {ok
            ? t`All mods present`
            : t`Has used ${plural(missing.length, { one: '# mod it lacks', other: '# mods it lacks' })}`}
        </Box>
        {ok ? null : <AddAll missing={missing} profile={profile} />}
      </Box>
      {ok ? null : (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, flexWrap: 'wrap' }}>
          <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>{t`Has used:`}</Typography>
          {missing.map((lack) => (
            <LackChip key={lack.uniqueId} fit={fit} lack={lack} profile={profile} game={game} />
          ))}
        </Box>
      )}
    </Box>
  )
}

export function SavesTab({ profile, game }: { profile: Profile; game: string }) {
  const { t } = useLingui()
  const { fits, status, error: detail, load } = useSaves()
  const { name } = profile
  let body: ReactNode = null
  if (status === 'error') {
    body = (
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Typography sx={{ fontSize: 13, color: 'error.main' }}>
          {t`Could not read your saves: ${detail}`}
        </Typography>
        <Button
          size="small"
          sx={nowrap}
          onClick={() => {
            load(game, profile.id, String(profile.updated)).catch(reportUnexpected)
          }}
        >
          {t`Retry`}
        </Button>
      </Box>
    )
  } else if (status === 'loading' && fits.length === 0) {
    body = <Typography sx={{ color: 'text.secondary' }}>{t`Reading your saves…`}</Typography>
  } else if (fits.length === 0) {
    body = <Typography sx={{ color: 'text.secondary' }}>{t`No saves found yet.`}</Typography>
  } else {
    body = fits.map((fit) => <SaveRow key={fit.folder} fit={fit} profile={profile} game={game} />)
  }
  return (
    <Box sx={{ flex: 1, minHeight: 0, overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
      <Typography sx={{ px: 2, pt: 1.5, pb: 0.75, fontSize: 13, lineHeight: 1.5 }}>
        {t`Mortar reads each save for the mods it has used. You pick the save in the game; this is how well each one fits ${name}.`}
      </Typography>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, px: 2, pt: 0.75, pb: 2 }}>
        {body}
      </Box>
    </Box>
  )
}
