import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, IconButton, Tooltip, Typography } from '@mui/material'
import { Browser } from '@wailsio/runtime'
import {
  CalendarDays,
  CircleCheck,
  Clock,
  Coins,
  ExternalLink,
  Flower2,
  History,
  Leaf,
  Plus,
  Power,
  Snowflake,
  Sprout,
  Sun,
  TriangleAlert,
  X,
} from 'lucide-react'
import { type ReactNode, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import type {
  Fit,
  Lack,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/models.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useLocked } from '../mods/useLocked.ts'
import { download, type Want } from '../queue/actions.ts'
import { useQueue } from '../queue/store.ts'
import { pendingFor } from '../queue/totals.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { TipBanner } from '../tips/TipBanner.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { BackupsDialog } from './BackupsDialog.tsx'
import { goldText, hoursPlayed } from './card.ts'
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
      latest: true,
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
  let plusTitle = t`Add to this profile`
  if (queued) {
    plusTitle = t`Queued`
  }
  if (locked) {
    plusTitle = t`Stop the game to change mods.`
  }
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
        <Tooltip title={plusTitle}>
          <span>
            <IconButton
              size="small"
              disabled={queued || locked}
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
  )
}

// Each season's tile colour and icon, solid so the card reads at a glance.
const SEASON_STYLE = [
  { color: '#4f9e52', Icon: Flower2 },
  { color: '#d29a22', Icon: Sun },
  { color: '#c0612b', Icon: Leaf },
  { color: '#5e8fb8', Icon: Snowflake },
] as const

function Stat({ icon, children }: { icon: ReactNode; children: ReactNode }) {
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 0.75,
        fontSize: 13,
        color: 'rgba(255,255,255,0.85)',
        ...nowrap,
      }}
    >
      <Box component="span" sx={{ display: 'flex', color: 'text.secondary' }}>
        {icon}
      </Box>
      {children}
    </Box>
  )
}

function useFarmKind(which: number): string {
  const { t } = useLingui()
  const kinds = [
    t`Standard farm`,
    t`Riverland farm`,
    t`Forest farm`,
    t`Hill-top farm`,
    t`Wilderness farm`,
    t`Four Corners farm`,
    t`Beach farm`,
    t`Meadowlands farm`,
  ]
  return kinds[which] ?? ''
}

function FitStatus({ missing }: { missing: number }) {
  const { t } = useLingui()
  const ok = missing === 0
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 0.5,
        px: 1.25,
        py: '3px',
        borderRadius: '12px',
        fontSize: 12,
        fontWeight: 700,
        bgcolor: ok ? 'rgba(12,223,100,0.16)' : 'rgba(243,180,22,0.18)',
        color: ok ? 'success.main' : 'warning.main',
        ...nowrap,
      }}
    >
      {ok ? <CircleCheck size={13} /> : <TriangleAlert size={13} />}
      {ok
        ? t`All mods present`
        : t`Has used ${plural(missing, { one: '# mod it lacks', other: '# mods it lacks' })}`}
    </Box>
  )
}

function SaveRow({ fit, profile, game }: { fit: Fit; profile: Profile; game: string }) {
  const { t } = useLingui()
  const seasons = [t`Spring`, t`Summer`, t`Fall`, t`Winter`]
  const missing = fit.missing ?? []
  const style = SEASON_STYLE[fit.season] ?? SEASON_STYLE[0]
  const kind = useFarmKind(fit.whichFarm)
  const hours = hoursPlayed(fit.millisecondsPlayed)
  const subtitle = [fit.farmer, kind].filter(Boolean).join(' · ')
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 1.25,
        p: 1.75,
        bgcolor: 'rgba(50,50,60,0.78)',
        border: '1px solid rgba(255,255,255,0.06)',
        borderRadius: '8px',
        '&:hover': { borderColor: 'rgba(255,255,255,0.16)' },
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
        <Box
          aria-hidden={true}
          sx={{
            width: 52,
            height: 52,
            flexShrink: 0,
            display: 'grid',
            placeItems: 'center',
            borderRadius: '10px',
            bgcolor: style.color,
            color: '#ffffff',
          }}
        >
          <style.Icon size={26} />
        </Box>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography noWrap={true} sx={{ fontSize: 17, fontWeight: 700 }}>
            {fit.farm || fit.folder}
          </Typography>
          <Typography noWrap={true} sx={{ fontSize: 13, color: 'text.secondary' }}>
            {subtitle}
          </Typography>
        </Box>
      </Box>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', columnGap: 2, rowGap: 0.75 }}>
        {fit.day > 0 ? (
          <Stat icon={<CalendarDays size={14} />}>
            {t`${seasons[fit.season] ?? ''} ${fit.day}, year ${fit.year}`}
          </Stat>
        ) : null}
        {hours > 0 ? <Stat icon={<Clock size={14} />}>{t`${hours}h played`}</Stat> : null}
        {fit.money ? <Stat icon={<Coins size={14} />}>{goldText(fit.money)}</Stat> : null}
      </Box>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          pt: 1,
          borderTop: '1px solid rgba(255,255,255,0.08)',
        }}
      >
        <Typography
          sx={{ flex: 1, minWidth: 0, fontSize: 12, color: 'text.secondary' }}
          noWrap={true}
        >
          {t`Last played ${formatWhen(fit.played)}`}
        </Typography>
        <FitStatus missing={missing.length} />
        {missing.length === 0 ? null : <AddAll missing={missing} profile={profile} />}
      </Box>
      {missing.length === 0 ? null : (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, flexWrap: 'wrap' }}>
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
  const [backupsOpen, setBackupsOpen] = useState(false)
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
    body = (
      <EmptyState icon={<Sprout size={40} aria-hidden={true} />} title={t`No saves yet`}>
        {t`Play this profile and start a farm. Each save shows here with how well it fits ${name}, so you know which mods it needs.`}
      </EmptyState>
    )
  } else {
    body = fits.map((fit) => <SaveRow key={fit.folder} fit={fit} profile={profile} game={game} />)
  }
  return (
    <Box sx={{ flex: 1, minHeight: 0, overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
      <TipBanner tip="saves">
        {t`Saves stay in one folder for every profile. This tab shows how well each save fits this one.`}
      </TipBanner>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 2, pt: 1.5, pb: 0.75 }}>
        <Typography sx={{ flex: 1, minWidth: 0, fontSize: 13, lineHeight: 1.5 }}>
          {t`Mortar reads each save for the mods it has used. You pick the save in the game; this is how well each one fits ${name}.`}
        </Typography>
        <Button
          size="small"
          startIcon={<History size={14} />}
          onClick={() => setBackupsOpen(true)}
          sx={nowrap}
        >
          {t`Save backups`}
        </Button>
      </Box>
      <Box
        sx={{
          flex: fits.length > 0 ? undefined : 1,
          display: fits.length > 0 ? 'grid' : 'flex',
          gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))',
          flexDirection: 'column',
          gap: 1.25,
          px: 2,
          pt: 0.75,
          pb: 1.5,
        }}
      >
        {body}
      </Box>
      <BackupsDialog open={backupsOpen} onClose={() => setBackupsOpen(false)} />
    </Box>
  )
}
