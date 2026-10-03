import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, IconButton, Tooltip, Typography } from '@mui/material'
import {
  Archive,
  CircleCheck,
  Flower2,
  FolderOpen,
  Leaf,
  Snowflake,
  Sun,
  TriangleAlert,
  UserPlus,
} from 'lucide-react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import type { Fit } from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/models.ts'
import {
  CreateBackup,
  OpenSaveFolder,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/service.ts'
import { useLocked } from '../mods/useLocked.ts'
import { useProfiles } from '../profiles/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { errorDetails, errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { useSaveBackups } from './backups.ts'
import { newProfileFromSave } from './recordedActions.ts'
import { SaveDetails } from './SaveDetails.tsx'

const nowrap = { whiteSpace: 'nowrap' } as const

const SEASON_STYLE = [
  { color: 'success.main', Icon: Flower2 },
  { color: 'warning.main', Icon: Sun },
  { color: 'error.main', Icon: Leaf },
  { color: 'info.main', Icon: Snowflake },
] as const

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
        color: ok ? 'success.main' : 'warning.main',
        bgcolor: 'var(--mortar-overlay-30)',
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

function SaveButtons({ fit, game, label }: { fit: Fit; game: string; label: string }) {
  const { t } = useLingui()
  const [backingUp, runBackup] = usePending()
  const [creating, runCreate] = usePending()
  const locked = useLocked()
  const canFromSave = (fit.lastMods ?? []).length > 0
  return (
    <>
      <DisabledReason title={t`This save has no recorded mod list yet.`} disabled={!canFromSave}>
        <Tooltip title={t`New profile from this save`}>
          <span>
            <IconButton
              size="small"
              disabled={!canFromSave || creating || locked}
              aria-label={t`New profile from this save`}
              onClick={() => runCreate(() => newProfileFromSave(game, fit.folder))}
            >
              <UserPlus size={16} />
            </IconButton>
          </span>
        </Tooltip>
      </DisabledReason>
      <Tooltip title={t`Back up now`}>
        <span>
          <IconButton
            size="small"
            disabled={backingUp}
            aria-label={t`Back up ${label} now`}
            onClick={() => {
              runBackup(async () => {
                try {
                  await CreateBackup(fit.folder)
                  await useSaveBackups.getState().load()
                  useToasts.getState().push({ kind: 'success', title: t`Backed up ${label}` })
                } catch (e) {
                  useToasts.getState().push({
                    kind: 'error',
                    title: t`Could not back up ${label}`,
                    body: errorMessage(e),
                    detail: errorDetails(e),
                  })
                }
              })
            }}
          >
            <Archive size={16} />
          </IconButton>
        </span>
      </Tooltip>
      <Tooltip title={t`Open save folder`}>
        <IconButton
          size="small"
          aria-label={t`Open the folder of ${label}`}
          onClick={() => {
            OpenSaveFolder(fit.folder).catch(() => undefined)
          }}
        >
          <FolderOpen size={16} />
        </IconButton>
      </Tooltip>
    </>
  )
}

export function SaveRow({ fit, profile, game }: { fit: Fit; profile: Profile; game: string }) {
  const { t } = useLingui()
  const missing = fit.missing ?? []
  const style = SEASON_STYLE[fit.season] ?? SEASON_STYLE[0]
  const kind = useFarmKind(fit.whichFarm)
  const subtitle = [fit.farmer, kind].filter(Boolean).join(' · ')
  const label = fit.farm || fit.folder
  const lastProfile = useProfiles.getState().profiles.find((p) => p.id === fit.lastProfileId)
  const lastGone = Boolean(fit.lastProfileId) && lastProfile === undefined
  const lastName = lastProfile?.name ?? (lastGone ? t`a deleted profile` : '')
  const lastLine = lastName === '' ? '' : t`Last played with ${lastName}`
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 1.25,
        p: 1.75,
        bgcolor: 'var(--mortar-paper-78)',
        border: '1px solid var(--mortar-hairline-faint)',
        borderRadius: '8px',
        '&:hover': { borderColor: 'var(--mortar-hairline-16)' },
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
            color: 'var(--mortar-ink)',
          }}
        >
          <style.Icon size={26} />
        </Box>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography noWrap={true} title={label} sx={{ fontSize: 17, fontWeight: 700 }}>
            {label}
          </Typography>
          <Typography noWrap={true} sx={{ fontSize: 13, color: 'text.secondary' }}>
            {subtitle}
          </Typography>
        </Box>
        <SaveButtons fit={fit} game={game} label={label} />
      </Box>
      <SaveDetails
        fit={fit}
        profile={profile}
        game={game}
        lastLine={lastLine}
        lastGone={lastGone}
        status={<FitStatus missing={missing.length} />}
      />
    </Box>
  )
}
