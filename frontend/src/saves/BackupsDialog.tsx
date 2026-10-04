import { i18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Menu,
  Tooltip,
  Typography,
} from '@mui/material'
import { FolderOpen, Pin, PinOff, RotateCcw } from 'lucide-react'
import { useEffect, useState } from 'react'
import type {
  Backup,
  Snap,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/backup/models.ts'
import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { When } from '../i18n/When.tsx'
import { useLaunch } from '../launch/store.ts'
import { paper } from '../mods/paper.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { useSaveBackups } from './backups.ts'
import { useSaves } from './store.ts'

const nowrap = { whiteSpace: 'nowrap' } as const

function overwriteMessage(snaps: Snap[], have: Set<string>): string {
  const names = snaps.map((s) => s.farm || s.folder)
  const hit = snaps.filter((s) => have.has(s.folder)).map((s) => s.farm || s.folder)
  if (hit.length === 0) {
    return i18n._(
      msg`None of ${names.join(', ')} are in Saves yet; they will be added. A backup of the current Saves folder is made first.`,
    )
  }
  return i18n._(
    msg`${hit.join(', ')} will be overwritten. A backup of the current Saves folder is made first.`,
  )
}

function causeLabel(b: Backup, profileName: (id: string) => string): string {
  if (b.kind === 'update' && b.profile) {
    return i18n._(msg`Before updating ${profileName(b.profile)}`)
  }
  if (b.kind === 'restore') {
    return i18n._(msg`Before a restore`)
  }
  if (b.kind === 'launch') {
    return i18n._(msg`Before playing`)
  }
  if (b.kind === 'manual') {
    return i18n._(msg`Manual`)
  }
  return i18n._(msg`Unknown`)
}

function BackupRow({
  backup,
  busyGame,
  pending,
  profileName,
  onRestore,
}: {
  backup: Backup
  busyGame: boolean
  pending: boolean
  profileName: (id: string) => string
  onRestore: (el: HTMLElement) => void
}) {
  const { t } = useLingui()
  const when = backup.at > 0 ? <When value={backup.at} withTime={true} /> : backup.name
  const farms = (backup.saves ?? []).map((s) => s.farm || s.folder).join(', ')
  const meta = farms
    ? t`${causeLabel(backup, profileName)} · ${formatBytes(backup.size)} · ${farms}`
    : t`${causeLabel(backup, profileName)} · ${formatBytes(backup.size)}`
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        px: 1.5,
        py: 1.25,
        bgcolor: 'var(--mortar-paper-78)',
        borderRadius: '6px',
      }}
    >
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography noWrap={true} sx={{ fontSize: 14, fontWeight: 600 }}>
          {when}
        </Typography>
        <Typography
          noWrap={true}
          title={backup.name}
          sx={{ fontSize: 13, color: 'text.secondary' }}
        >
          {meta}
        </Typography>
      </Box>
      <Tooltip title={busyGame ? t`Stop the game to restore saves.` : t`Restore`}>
        <span>
          <IconButton
            size="small"
            disabled={busyGame || pending}
            aria-label={t`Restore ${when}`}
            onClick={(e) => {
              onRestore(e.currentTarget)
            }}
          >
            <RotateCcw size={16} />
          </IconButton>
        </span>
      </Tooltip>
    </Box>
  )
}

export function BackupsDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  const { items, status, error, load, restore, setPinned, openFolder } = useSaveBackups()
  const profiles = useProfiles((s) => s.profiles)
  const fits = useSaves((s) => s.fits)
  const busyGame = useLaunch(
    (s) => s.starting || s.status?.state === State.Launching || s.status?.state === State.Running,
  )
  const [pending, run] = usePending()
  const [menu, setMenu] = useState<{ backup: Backup; el: HTMLElement } | null>(null)
  const [confirm, setConfirm] = useState<{ backup: Backup; snaps: Snap[] } | null>(null)

  useEffect(() => {
    if (open) {
      load().catch(reportUnexpected)
    }
  }, [open, load])

  const profileName = (id: string) => profiles.find((p) => p.id === id)?.name ?? id

  return (
    <>
      <Dialog
        open={open}
        onClose={onClose}
        transitionDuration={0}
        slotProps={{ paper }}
        maxWidth="sm"
        fullWidth={true}
      >
        <DialogTitle>{t`Save backups`}</DialogTitle>
        <DialogContent>
          {status === 'error' ? (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <Typography sx={{ fontSize: 13, color: 'error.main' }}>
                {t`Could not list backups: ${error}`}
              </Typography>
              <Button size="small" sx={nowrap} onClick={() => load().catch(reportUnexpected)}>
                {t`Retry`}
              </Button>
            </Box>
          ) : null}
          {status === 'loading' && items.length === 0 ? (
            <LoadingRow>{t`Reading backups…`}</LoadingRow>
          ) : null}
          {status === 'ready' && items.length === 0 ? (
            <EmptyState
              compact={true}
              icon={<FolderOpen size={28} />}
              title={t`No save backups yet.`}
            >{t`Backups appear when Mortar creates a save backup.`}</EmptyState>
          ) : null}
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {items.map((b) => (
              <BackupRow
                key={b.name}
                backup={b}
                busyGame={busyGame}
                pending={pending}
                profileName={profileName}
                onRestore={(el) => {
                  setMenu({ backup: b, el })
                }}
              />
            ))}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button
            startIcon={<FolderOpen size={16} />}
            onClick={() => {
              openFolder().catch(reportUnexpected)
            }}
            sx={nowrap}
          >
            {t`Open backups folder`}
          </Button>
          <Button onClick={onClose} sx={nowrap}>
            {t`Close`}
          </Button>
        </DialogActions>
      </Dialog>
      <Menu
        open={menu !== null}
        anchorEl={menu?.el}
        onClose={() => {
          setMenu(null)
        }}
      >
        <MenuAction
          icon={menu?.backup.pinned ? <PinOff size={16} /> : <Pin size={16} />}
          label={menu?.backup.pinned ? t`Unkeep` : t`Keep`}
          onClick={() => {
            if (menu) {
              const { backup } = menu
              setPinned(backup.name, !backup.pinned).catch(reportUnexpected)
            }
            setMenu(null)
          }}
        />
        <MenuAction
          icon={<RotateCcw size={16} />}
          label={t`Restore all`}
          onClick={() => {
            if (menu) {
              setConfirm({ backup: menu.backup, snaps: menu.backup.saves ?? [] })
            }
            setMenu(null)
          }}
        />
        {(menu?.backup.saves ?? []).map((snap) => (
          <MenuAction
            key={snap.folder}
            icon={<RotateCcw size={16} />}
            label={t`Restore ${snap.farm || snap.folder}`}
            onClick={() => {
              if (menu) {
                setConfirm({ backup: menu.backup, snaps: [snap] })
              }
              setMenu(null)
            }}
          />
        ))}
      </Menu>
      <ConfirmDialog
        open={confirm !== null}
        title={t`Restore this backup?`}
        confirmLabel={t`Restore`}
        busy={pending || busyGame}
        onCancel={() => {
          setConfirm(null)
        }}
        onConfirm={() => {
          if (!confirm) {
            return
          }
          const { backup, snaps } = confirm
          run(async () => {
            await restore(
              backup.name,
              snaps.length === (backup.saves ?? []).length ? [] : snaps.map((s) => s.folder),
            )
            setConfirm(null)
          })
        }}
      >
        <Typography sx={{ fontSize: 14 }}>
          {overwriteMessage(confirm?.snaps ?? [], new Set((fits ?? []).map((f) => f.folder)))}
        </Typography>
      </ConfirmDialog>
    </>
  )
}
