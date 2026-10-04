import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button, TextField } from '@mui/material'
import { useEffect, useState } from 'react'
import type { BackupsUsage } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/models.ts'
import {
  BackupsUsage as LoadBackupsUsage,
  TrimBackups,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/service.ts'
import { formatBytes } from '../../i18n/bytes.ts'
import { useProfiles } from '../../profiles/store.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { usePending } from '../../toasts/usePending.ts'
import { SettingRow } from '../SettingsSection.tsx'

const MIN_KEEP = 1

export function TrimDialog({
  title,
  body,
  fieldLabel,
  defaultKeep,
  open,
  onClose,
  onTrim,
  busy,
}: {
  title: string
  body: string
  fieldLabel: string
  defaultKeep: number
  open: boolean
  onClose: () => void
  onTrim: (keep: number) => void
  busy: boolean
}) {
  const { t } = useLingui()
  const [draft, setDraft] = useState(String(defaultKeep))
  useEffect(() => {
    if (open) {
      setDraft(String(defaultKeep))
    }
  }, [open, defaultKeep])
  const keep = Number(draft)
  const valid = Number.isInteger(keep) && keep >= MIN_KEEP
  return (
    <ConfirmDialog
      open={open}
      color="error"
      fieldFocus={true}
      busy={busy}
      title={title}
      body={body}
      confirmLabel={t`Trim`}
      confirmDisabled={!valid}
      onCancel={onClose}
      onConfirm={() => onTrim(keep)}
    >
      <form
        onSubmit={(e) => {
          e.preventDefault()
          if (valid && !busy) {
            onTrim(keep)
          }
        }}
      >
        <TextField
          type="number"
          size="small"
          label={fieldLabel}
          value={draft}
          error={!valid}
          helperText={valid ? '' : t`Enter 1 or more`}
          autoFocus={true}
          onChange={(e) => setDraft(e.target.value)}
          slotProps={{ htmlInput: { min: MIN_KEEP, step: 1 } }}
          sx={{ mt: 1.5 }}
        />
      </form>
    </ConfirmDialog>
  )
}

export function BackupsUsageRow() {
  const { t } = useLingui()
  const gameId = useProfiles((s) => s.game?.id)
  const [usage, setUsage] = useState<BackupsUsage | null>(null)
  const [open, setOpen] = useState(false)
  const [pending, run] = usePending()
  useEffect(() => {
    if (gameId) {
      LoadBackupsUsage(gameId).then(setUsage).catch(reportUnexpected)
    }
  }, [gameId])
  if (!gameId || usage === null) {
    return null
  }
  const trim = (keep: number) =>
    run(
      async () => {
        const res = await TrimBackups(gameId, keep)
        setOpen(false)
        useToasts.getState().push({
          kind: 'success',
          title:
            res.removed === 0
              ? t`No backups to remove`
              : t`${plural(res.removed, { one: 'Removed # backup', other: 'Removed # backups' })}, freed ${formatBytes(res.freedBytes)}`,
        })
        setUsage(await LoadBackupsUsage(gameId))
      },
      { errorTitle: t`Could not trim save backups` },
    )
  return (
    <SettingRow label={t`Save backups: ${formatBytes(usage.totalBytes)}`}>
      <Button variant="outlined" disabled={usage.totalBytes === 0} onClick={() => setOpen(true)}>
        {t`Trim…`}
      </Button>
      <TrimDialog
        title={t`Trim save backups`}
        body={t`Keep the newest backups of each save and delete the rest. Pinned backups are never removed.`}
        fieldLabel={t`Backups to keep per save`}
        defaultKeep={5}
        open={open}
        busy={pending}
        onClose={() => setOpen(false)}
        onTrim={trim}
      />
    </SettingRow>
  )
}
