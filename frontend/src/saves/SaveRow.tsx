import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import {
  Archive,
  CircleCheck,
  CircleHelp,
  Flower2,
  FolderOpen,
  Leaf,
  Save,
  Snowflake,
  Sun,
  TriangleAlert,
  UserPlus,
} from 'lucide-react'
import type { Backup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/backup/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Fit } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/models.ts'
import {
  CreateBackup,
  OpenSaveFolder,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { useLocked } from '../mods/useLocked.ts'
import { useProfiles } from '../profiles/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { useSaveBackups } from './backups.ts'
import { newProfileFromSave } from './recordedActions.ts'
import { SaveBackupsButton } from './SaveBackupsButton.tsx'
import { SaveDetails } from './SaveDetails.tsx'
import { SaveGapLine } from './SaveGapLine.tsx'
import { saveKind, saveName } from './saveName.ts'
import { useSaves } from './store.ts'

const nowrap = { whiteSpace: 'nowrap' } as const

const SEASON_STYLE = [
  { color: 'success.main', Icon: Flower2 },
  { color: 'warning.main', Icon: Sun },
  { color: 'error.main', Icon: Leaf },
  { color: 'info.main', Icon: Snowflake },
] as const

// A save without Stardew's calendar (a Lethal Company slot) gets no season colour.
const FILE_STYLE = { color: 'text.secondary', Icon: Save } as const

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

export function FitStatus({ missing, unrecorded }: { missing: number; unrecorded: boolean }) {
  const { t } = useLingui()
  let color = 'warning.main'
  let icon = <TriangleAlert size={13} />
  let text = t`Has used ${plural(missing, { one: '# mod it lacks', other: '# mods it lacks' })}`
  if (unrecorded) {
    color = 'text.secondary'
    icon = <CircleHelp size={13} />
    text = t`Mods not recorded`
  } else if (missing === 0) {
    color = 'success.main'
    icon = <CircleCheck size={13} />
    text = t`All mods present`
  }
  return (
    <Box
      title={unrecorded ? t`This game's saves do not record which mods they used.` : undefined}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 0.5,
        px: space.gap,
        py: '3px',
        borderRadius: '12px',
        fontSize: 12,
        fontWeight: 700,
        color,
        bgcolor: 'var(--mortar-overlay-30)',
        ...nowrap,
      }}
    >
      {icon}
      {text}
    </Box>
  )
}

export function SaveButtons({
  fit,
  game,
  profileId,
  label,
  backups,
  onBackupsChanged,
}: {
  fit: Fit
  game: string
  profileId: string
  label: string
  backups: Backup[] | null
  onBackupsChanged: () => Promise<void>
}) {
  const { t } = useLingui()
  const [backingUp, runBackup] = usePending()
  const [creating, runCreate] = usePending()
  const locked = useLocked()
  const canFromSave = (fit.lastMods ?? []).length > 0
  return (
    <>
      <SaveBackupsButton
        folder={fit.folder}
        label={label}
        profile={profileId}
        backups={backups}
        onChanged={onBackupsChanged}
      />
      {/* A save that never records its mods (Lethal Company's) can never seed a profile. */}
      {fit.unrecorded ? null : (
        <DisabledReason
          title={
            locked ? t`Stop the game to change mods.` : t`This save has no recorded mod list yet.`
          }
          disabled={!canFromSave || locked}
        >
          <TipIconButton
            label={t`New profile from this save`}
            disabled={!canFromSave || creating || locked}
            onClick={() => runCreate(() => newProfileFromSave(game, fit.folder))}
          >
            <UserPlus size={16} />
          </TipIconButton>
        </DisabledReason>
      )}
      <TipIconButton
        label={t`Back up ${label} now`}
        disabled={backingUp}
        onClick={() => {
          runBackup(
            async () => {
              if (!(await CreateBackup(game, profileId, fit.folder))) {
                useToasts
                  .getState()
                  .push({ kind: 'info', title: t`${label} is no longer in the saves folder` })
                // The save left the folder since the list was read, so the list drops its row.
                await useSaves.getState().reload()
                return
              }
              await Promise.all([useSaveBackups.getState().reload(), onBackupsChanged()])
              useToasts.getState().push({ kind: 'success', title: t`Backed up ${label}` })
            },
            { errorTitle: t`Could not back up ${label}` },
          )
        }}
      >
        <Archive size={16} />
      </TipIconButton>
      <TipIconButton
        label={t`Open the folder of ${label}`}
        onClick={() => {
          OpenSaveFolder(game, profileId, fit.folder).catch(reportUnexpected)
        }}
      >
        <FolderOpen size={16} />
      </TipIconButton>
    </>
  )
}

export function SaveRow({
  fit,
  profile,
  game,
  backups,
  onBackupsChanged,
}: {
  fit: Fit
  profile: Profile
  game: string
  backups: Backup[] | null
  onBackupsChanged: () => Promise<void>
}) {
  const { t } = useLingui()
  const missing = fit.missing ?? []
  const style = fit.unrecorded ? FILE_STYLE : (SEASON_STYLE[fit.season] ?? SEASON_STYLE[0])
  const kind = useFarmKind(fit.whichFarm)
  const subtitle = [fit.farmer, kind, saveKind(fit.folder)].filter(Boolean).join(' · ')
  const label = saveName(fit)
  const lastProfile = useProfiles.getState().profiles.find((p) => p.id === fit.lastProfileId)
  const lastGone = Boolean(fit.lastProfileId) && lastProfile === undefined
  const lastName = lastProfile?.name ?? (lastGone ? t`a deleted profile` : '')
  const lastLine = lastName === '' ? '' : t`Last played with ${{ profile: lastName }}`
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: space.gap,
        p: space.pad,
        bgcolor: 'var(--mortar-paper-78)',
        border: '1px solid var(--mortar-hairline-faint)',
        borderRadius: '8px',
        '&:hover': { borderColor: 'var(--mortar-hairline-16)' },
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
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
          <Typography title={subtitle} noWrap={true} sx={{ fontSize: 13, color: 'text.secondary' }}>
            {subtitle}
          </Typography>
        </Box>
        <SaveButtons
          fit={fit}
          game={game}
          profileId={profile.id}
          label={label}
          backups={backups}
          onBackupsChanged={onBackupsChanged}
        />
      </Box>
      <SaveDetails
        fit={fit}
        profile={profile}
        game={game}
        lastLine={lastLine}
        lastGone={lastGone}
        status={<FitStatus missing={missing.length} unrecorded={fit.unrecorded} />}
      />
      <SaveGapLine
        key={`${profile.updated}-${fit.lastProfileAt}`}
        fit={fit}
        profile={profile}
        game={game}
      />
    </Box>
  )
}
