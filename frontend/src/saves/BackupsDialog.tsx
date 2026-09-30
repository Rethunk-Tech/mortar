import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Tooltip,
  Typography,
} from '@mui/material'
import { FolderOpen, RotateCcw } from 'lucide-react'
import { useEffect, useState } from 'react'
import type {
  Backup,
  Snap,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/backup/models.ts'
import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { useLaunch } from '../launch/store.ts'
import { paper } from '../mods/paper.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { formatBytes } from './backupFormat.ts'
import { useSaveBackups } from './backups.ts'
import { useSaves } from './store.ts'

const nowrap = { whiteSpace: 'nowrap' } as const

function overwriteMessage(
  t: ReturnType<typeof useLingui>['t'],
  snaps: Snap[],
  have: Set<string>,
): string {
  const names = snaps.map((s) => s.farm || s.folder)
  const hit = snaps.filter((s) => have.has(s.folder)).map((s) => s.farm || s.folder)
  if (hit.length === 0) {
    return t`None of ${names.join(', ')} are in Saves yet; they will be added. A backup of the current Saves folder is made first.`
  }
  return t`${hit.join(', ')} will be overwritten. A backup of the current Saves folder is made first.`
}

function causeLabel(
  t: ReturnType<typeof useLingui>['t'],
  b: Backup,
  profileName: (id: string) => string,
): string {
  if (b.kind === 'update' && b.profile) {
    return t`Before updating ${profileName(b.profile)}`
  }
  if (b.kind === 'restore') {
    return t`Before a restore`
  }
  return t`Unknown`
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
  const when = backup.at > 0 ? new Date(backup.at).toLocaleString() : backup.name
  const farms = (backup.saves ?? []).map((s) => s.farm || s.folder).join(', ')
  const meta = farms
    ? t`${causeLabel(t, backup, profileName)} · ${formatBytes(backup.size)} · ${farms}`
    : t`${causeLabel(t, backup, profileName)} · ${formatBytes(backup.size)}`
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        px: 1.5,
        py: 1.25,
        bgcolor: 'rgba(50,50,60,0.78)',
        borderRadius: '6px',
      }}
    >
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography noWrap={true} sx={{ fontSize: 14, fontWeight: 600 }}>
          {when}
        </Typography>
        <Typography noWrap={true} sx={{ fontSize: 13, color: 'text.secondary' }}>
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
  const { items, status, error, load, restore, openFolder } = useSaveBackups()
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
            <Typography sx={{ fontSize: 13, color: 'error.main' }}>
              {t`Could not list backups: ${error}`}
            </Typography>
          ) : null}
          {status === 'loading' && items.length === 0 ? (
            <Typography sx={{ color: 'text.secondary' }}>{t`Reading backups…`}</Typography>
          ) : null}
          {status === 'ready' && items.length === 0 ? (
            <Typography sx={{ color: 'text.secondary' }}>{t`No save backups yet.`}</Typography>
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
        <MenuItem
          onClick={() => {
            if (menu) {
              setConfirm({ backup: menu.backup, snaps: menu.backup.saves ?? [] })
            }
            setMenu(null)
          }}
        >
          <ListItemIcon>
            <RotateCcw size={16} />
          </ListItemIcon>
          <ListItemText>{t`Restore all`}</ListItemText>
        </MenuItem>
        {(menu?.backup.saves ?? []).map((snap) => (
          <MenuItem
            key={snap.folder}
            onClick={() => {
              if (menu) {
                setConfirm({ backup: menu.backup, snaps: [snap] })
              }
              setMenu(null)
            }}
          >
            <ListItemIcon>
              <RotateCcw size={16} />
            </ListItemIcon>
            <ListItemText>{t`Restore ${snap.farm || snap.folder}`}</ListItemText>
          </MenuItem>
        ))}
      </Menu>
      <Dialog
        open={confirm !== null}
        onClose={() => {
          setConfirm(null)
        }}
        transitionDuration={0}
        slotProps={{ paper }}
      >
        <DialogTitle>{t`Restore this backup?`}</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 14 }}>
            {overwriteMessage(t, confirm?.snaps ?? [], new Set((fits ?? []).map((f) => f.folder)))}
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => {
              setConfirm(null)
            }}
            sx={nowrap}
          >
            {t`Cancel`}
          </Button>
          <Button
            variant="contained"
            disabled={pending || busyGame}
            onClick={() => {
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
            sx={nowrap}
          >
            {t`Restore`}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  )
}
