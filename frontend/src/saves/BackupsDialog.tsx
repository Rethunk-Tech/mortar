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
  Menu,
  Tooltip,
  Typography,
} from '@mui/material'
import { FolderOpen, Pin, PinOff, RotateCcw } from 'lucide-react'
import { useEffect, useState } from 'react'
import type {
  Backup,
  Snap,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/backup/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { listNames } from '../i18n/list.ts'
import { When } from '../i18n/When.tsx'
import { useGameBusy } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { causeLabel, useSaveBackups } from './backups.ts'
import { saveName } from './saveName.ts'
import { useSaves } from './store.ts'

function overwriteMessage(snaps: Snap[], have: Set<string>): string {
  const names = snaps.map(saveName)
  const hit = snaps.filter((s) => have.has(s.folder)).map(saveName)
  if (hit.length === 0) {
    return i18n._(
      msg`None of ${listNames(names)} are in Saves yet; they will be added. The current Saves folder is backed up first.`,
    )
  }
  return i18n._(
    msg`${listNames(hit)} will be overwritten. A backup of the current Saves folder is made first.`,
  )
}

function BackupRow({
  backup,
  busyGame,
  pending,
  profiles,
  menuOpen,
  onRestore,
}: {
  backup: Backup
  busyGame: boolean
  pending: boolean
  profiles: readonly Profile[]
  menuOpen: boolean
  onRestore: (el: HTMLElement) => void
}) {
  const { t } = useLingui()
  const when = backup.at > 0 ? <When value={backup.at} withTime={true} /> : backup.name
  const farms = (backup.saves ?? []).map(saveName).join(', ')
  const meta = farms
    ? t`${causeLabel(backup, profiles)} · ${formatBytes(backup.size)} · ${farms}`
    : t`${causeLabel(backup, profiles)} · ${formatBytes(backup.size)}`
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
      {backup.pinned && (
        <Tooltip title={t`Kept forever`}>
          <Box
            component="span"
            role="img"
            aria-label={t`Kept forever`}
            sx={{ display: 'flex', color: 'text.secondary' }}
          >
            <Pin size={14} />
          </Box>
        </Tooltip>
      )}
      <TipIconButton
        label={busyGame ? t`Stop the game to restore saves.` : t`Restore ${{ label: when }}`}
        disabled={busyGame || pending}
        aria-haspopup="menu"
        aria-expanded={menuOpen}
        onClick={(e) => {
          onRestore(e.currentTarget)
        }}
      >
        <RotateCcw size={16} />
      </TipIconButton>
    </Box>
  )
}

export function BackupsDialog({
  open,
  onClose,
  game,
  profile,
}: {
  open: boolean
  onClose: () => void
  game: string
  profile: string
}) {
  const { t } = useLingui()
  const { items, status, error, load, restore, setPinned, openFolder } = useSaveBackups()
  const profiles = useProfiles((s) => s.profiles)
  const fits = useSaves((s) => s.fits)
  const busyGame = useGameBusy()
  const [pending, run] = usePending()
  const [menu, setMenu] = useState<{ backup: Backup; el: HTMLElement } | null>(null)
  const [confirm, setConfirm] = useState<{ backup: Backup; snaps: Snap[] } | null>(null)

  useEffect(() => {
    if (open) {
      load(game, profile).catch(reportUnexpected)
    }
  }, [open, load, game, profile])

  return (
    <>
      <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth={true}>
        <DialogTitle>{t`Save backups`}</DialogTitle>
        <DialogContent>
          {status === 'error' ? (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <Typography sx={{ fontSize: 13, color: 'error.main' }}>
                {t`Could not list backups: ${error}`}
              </Typography>
              <Button size="small" onClick={() => load(game, profile).catch(reportUnexpected)}>
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
            >{t`Mortar backs up your saves before playing and before updates. Use Back up now on a save to make one yourself.`}</EmptyState>
          ) : null}
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {items.map((b) => (
              <BackupRow
                key={b.name}
                backup={b}
                busyGame={busyGame}
                pending={pending}
                profiles={profiles}
                menuOpen={menu?.backup.name === b.name}
                onRestore={(el) => {
                  setMenu({ backup: b, el })
                }}
              />
            ))}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>{t`Close`}</Button>
          <Button
            startIcon={<FolderOpen size={16} />}
            onClick={() => {
              openFolder().catch(reportUnexpected)
            }}
          >
            {t`Open backups folder`}
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
          label={menu?.backup.pinned ? t`Allow cleanup` : t`Keep forever`}
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
            label={t`Restore ${{ label: saveName(snap) }}`}
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
