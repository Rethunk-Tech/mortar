import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { ChevronDown, ChevronRight, Pin, PinOff, RotateCcw, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import type { Backup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/backup/models.ts'
import {
  DeleteBackup,
  RestoreBackup,
  SaveBackups,
  SetBackupPinned,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { When } from '../i18n/When.tsx'
import { useGameBusy } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { causeLabel, useSaveBackups } from './backups.ts'
import { useSaves } from './store.ts'

interface Ask {
  kind: 'restore' | 'delete'
  backup: Backup
}

function Row({
  backup,
  busyGame,
  pending,
  onAsk,
  onPin,
}: {
  backup: Backup
  busyGame: boolean
  pending: boolean
  onAsk: (a: Ask) => void
  onPin: (b: Backup) => void
}) {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles)
  const cause = causeLabel(backup, (id) => profiles.find((p) => p.id === id)?.name ?? id)
  const size = formatBytes(backup.size)
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
      <Typography noWrap={true} sx={{ flex: 1, minWidth: 0, fontSize: 12 }}>
        {backup.at > 0 ? <When value={backup.at} withTime={true} /> : backup.name}
        <Box component="span" sx={{ color: 'text.secondary' }}>
          {t` · ${cause} · ${size}`}
        </Box>
      </Typography>
      <TipIconButton
        label={backup.pinned ? t`Allow cleanup` : t`Keep forever`}
        onClick={() => onPin(backup)}
      >
        {backup.pinned ? <PinOff size={14} /> : <Pin size={14} />}
      </TipIconButton>
      <TipIconButton
        label={busyGame ? t`Stop the game to restore saves.` : t`Restore`}
        disabled={busyGame || pending}
        onClick={() => onAsk({ kind: 'restore', backup })}
      >
        <RotateCcw size={14} />
      </TipIconButton>
      <TipIconButton
        label={backup.pinned ? t`Allow cleanup before deleting.` : t`Delete`}
        disabled={backup.pinned || pending}
        onClick={() => onAsk({ kind: 'delete', backup })}
      >
        <Trash2 size={14} />
      </TipIconButton>
    </Box>
  )
}

export function SaveBackupsSection({ folder, label }: { folder: string; label: string }) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState<Backup[] | null>(null)
  const [ask, setAsk] = useState<Ask | null>(null)
  const [pending, run] = usePending()
  const busyGame = useGameBusy()
  const load = useCallback(
    () =>
      SaveBackups(folder)
        .then((rows) => setItems(rows ?? []))
        .catch(reportUnexpected),
    [folder],
  )
  useEffect(() => {
    if (open) {
      load()
    }
  }, [open, load])
  const refresh = async () => {
    await load()
    await useSaveBackups.getState().load()
  }
  const when = ask ? formatWhen(ask.backup.at, { withTime: true }) : ''
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Button
        size="small"
        color="inherit"
        aria-expanded={open}
        startIcon={open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        onClick={() => setOpen(!open)}
        sx={{ alignSelf: 'flex-start', color: 'text.secondary' }}
      >
        {t`Backups`}
      </Button>
      {open && items === null ? <LoadingRow>{t`Reading backups…`}</LoadingRow> : null}
      {open && items?.length === 0 ? (
        <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
          {t`No backup contains this save yet.`}
        </Typography>
      ) : null}
      {open
        ? (items ?? []).map((b) => (
            <Row
              key={b.name}
              backup={b}
              busyGame={busyGame}
              pending={pending}
              onAsk={setAsk}
              onPin={(x) => {
                SetBackupPinned(x.name, !x.pinned).then(refresh).catch(reportUnexpected)
              }}
            />
          ))
        : null}
      <ConfirmDialog
        open={ask !== null}
        title={ask?.kind === 'delete' ? t`Delete this backup?` : t`Restore ${label} to ${when}?`}
        confirmLabel={ask?.kind === 'delete' ? t`Delete` : t`Restore`}
        color={ask?.kind === 'delete' ? 'error' : 'primary'}
        busy={pending || (ask?.kind === 'restore' && busyGame)}
        body={
          ask?.kind === 'delete'
            ? t`The backup from ${when} is removed for good, with every save in it.`
            : t`${label} is replaced by the copy in this backup. The current ${label} is backed up first.`
        }
        onCancel={() => setAsk(null)}
        onConfirm={() => {
          if (!ask) {
            return
          }
          const { kind, backup } = ask
          run(async () => {
            await (kind === 'delete'
              ? DeleteBackup(backup.name)
              : RestoreBackup(backup.name, [folder]))
            setAsk(null)
            await refresh()
            if (kind === 'restore') {
              await useSaves.getState().reload()
            }
          })
        }}
      />
    </Box>
  )
}
