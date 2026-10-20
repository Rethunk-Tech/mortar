import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Badge, Box, Popover, Typography } from '@mui/material'
import { History, Pin, PinOff, RotateCcw, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Backup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/backup/models.ts'
import type { Copy } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/saves/models.ts'
import {
  DeleteBackup,
  RestoreBackup,
  RestoreSaveCopy,
  SaveCopies,
  SetBackupPinned,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { When } from '../i18n/When.tsx'
import { useGameBusy } from '../launch/store.ts'
import { useCurrentGame } from '../nav/currentGame.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { causeLabel, useSaveBackups } from './backups.ts'
import { useSaves } from './store.ts'

type Ask = { kind: 'restore' | 'delete'; backup: Backup } | { kind: 'copy'; copy: Copy }

// One copy the game keeps of the save itself (Valheim's automatic world backups, The Sims 4's rollback copies).
function CopyRow({
  copy,
  busyGame,
  pending,
  onAsk,
}: {
  copy: Copy
  busyGame: boolean
  pending: boolean
  onAsk: (a: Ask) => void
}) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
      <Typography noWrap={true} title={copy.name} sx={{ flex: 1, minWidth: 0, fontSize: 12 }}>
        <When value={copy.at} withTime={true} />
        <Box component="span" sx={{ color: 'text.secondary' }}>
          {t` · Kept by the game · ${formatBytes(copy.size)}`}
        </Box>
      </Typography>
      <TipIconButton
        label={busyGame ? t`Stop the game to restore saves.` : t`Restore`}
        disabled={busyGame || pending}
        onClick={() => onAsk({ kind: 'copy', copy })}
      >
        <RotateCcw size={14} />
      </TipIconButton>
    </Box>
  )
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
  const cause = causeLabel(backup, profiles)
  const size = formatBytes(backup.size)
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
      <Typography noWrap={true} sx={{ flex: 1, minWidth: 0, fontSize: 12 }}>
        {backup.at > 0 ? <When value={backup.at} withTime={true} /> : backup.name}
        <Box component="span" sx={{ color: 'text.secondary' }}>
          {backup.pinned ? t` · ${cause} · ${size} · Kept` : t` · ${cause} · ${size}`}
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

// A save's backups: an icon button with the count, opening a popover with the list, whose rows keep, restore and
// delete (each restore and delete asks first).
export function SaveBackupsButton({
  folder,
  label,
  profile,
  backups,
  onChanged,
}: {
  folder: string
  label: string
  profile: string
  // Null until the page has read the backups.
  backups: Backup[] | null
  onChanged: () => Promise<void>
}) {
  const { t, i18n } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [ask, setAsk] = useState<Ask | null>(null)
  const [pending, run] = usePending()
  const busyGame = useGameBusy()
  const game = useCurrentGame()
  const [copies, setCopies] = useState<Copy[]>([])
  useEffect(() => {
    if (anchor) {
      SaveCopies(game, profile, folder)
        .then((c) => setCopies(c ?? []))
        .catch(() => setCopies([]))
    }
  }, [anchor, game, profile, folder])
  const count = backups?.length ?? 0
  // The name starts with the badge's number, as a label that names what is shown must.
  const tip =
    count > 0 ? i18n._(plural(count, { one: '# backup', other: '# backups' })) : t`No backups yet`
  const refresh = async () => {
    await onChanged()
    await useSaveBackups.getState().reload()
  }
  let when = ''
  if (ask) {
    when = formatWhen(ask.kind === 'copy' ? ask.copy.at : ask.backup.at, { withTime: true })
  }
  let body = t`${label} is replaced by the copy in this backup. The current ${label} is backed up first.`
  if (ask?.kind === 'delete') {
    body = t`The backup from ${when} is deleted for good, with every save in it.`
  } else if (ask?.kind === 'copy') {
    body = t`${label} is replaced by the game's own copy from ${when}. The current ${label} is backed up first.`
  }
  return (
    <>
      <TipIconButton
        label={tip}
        aria-haspopup="dialog"
        aria-expanded={anchor !== null}
        onClick={(e) => setAnchor(e.currentTarget)}
      >
        <Badge badgeContent={count} color="primary" max={99} overlap="circular">
          <History size={16} />
        </Badge>
      </TipIconButton>
      <Popover
        open={anchor !== null}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
        transformOrigin={{ vertical: 'top', horizontal: 'left' }}
        slotProps={{ paper: { role: 'dialog', 'aria-label': t`Backups of ${label}` } }}
      >
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            gap: 0.5,
            p: space.pad,
            width: 420,
            maxWidth: '90vw',
          }}
        >
          {backups === null ? <LoadingRow>{t`Reading backups…`}</LoadingRow> : null}
          {backups?.length === 0 && copies.length === 0 ? (
            <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
              {t`No backup contains this save yet.`}
            </Typography>
          ) : null}
          {(backups ?? []).map((b) => (
            <Row
              key={b.name}
              backup={b}
              busyGame={busyGame}
              pending={pending}
              onAsk={setAsk}
              onPin={(x) => {
                SetBackupPinned(game, x.name, !x.pinned).then(refresh).catch(reportUnexpected)
              }}
            />
          ))}
          {copies.map((c) => (
            <CopyRow key={c.name} copy={c} busyGame={busyGame} pending={pending} onAsk={setAsk} />
          ))}
        </Box>
      </Popover>
      <ConfirmDialog
        open={ask !== null}
        title={ask?.kind === 'delete' ? t`Delete this backup?` : t`Restore ${label} to ${when}?`}
        confirmLabel={ask?.kind === 'delete' ? t`Delete` : t`Restore`}
        color={ask?.kind === 'delete' ? 'error' : 'primary'}
        busy={pending || (ask?.kind !== 'delete' && busyGame)}
        body={body}
        onCancel={() => setAsk(null)}
        onConfirm={() => {
          if (!ask) {
            return
          }
          run(async () => {
            if (ask.kind === 'copy') {
              await RestoreSaveCopy(game, profile, folder, ask.copy.name)
            } else if (ask.kind === 'delete') {
              await DeleteBackup(game, ask.backup.name)
            } else {
              await RestoreBackup(game, profile, ask.backup.name, [folder])
            }
            setAsk(null)
            await refresh()
            if (ask.kind !== 'delete') {
              await useSaves.getState().reload()
            }
          })
        }}
      />
    </>
  )
}
