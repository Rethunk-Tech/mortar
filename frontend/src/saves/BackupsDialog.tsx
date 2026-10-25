import { i18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Menu, Typography } from '@mui/material'
import { FolderOpen, Pin, RotateCcw } from 'lucide-react'
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
import { AppDialog, DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { SkeletonRows } from '../shell/SkeletonRows.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { space } from '../theme/density.ts'
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
      msg`None of ${{ names: listNames(names) }} are in Saves yet; they will be added. The current Saves folder is backed up first.`,
    )
  }
  return i18n._(
    msg`${{ names: listNames(hit) }} will be overwritten. A backup of the current Saves folder is made first.`,
  )
}

function BackupRow({
  backup,
  busyGame,
  pending,
  profiles,
  menuOpen,
  onRestore,
  onPin,
}: {
  backup: Backup
  busyGame: boolean
  pending: boolean
  profiles: readonly Profile[]
  menuOpen: boolean
  onRestore: (el: HTMLElement) => void
  onPin: () => void
}) {
  const { t } = useLingui()
  const when = backup.at > 0 ? <When value={backup.at} withTime={true} /> : backup.name
  const farms = (backup.saves ?? []).map(saveName).join(', ')
  const meta = [causeLabel(backup, profiles), formatBytes(backup.size), farms]
    .filter(Boolean)
    .join(' · ')
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: space.gap,
        px: space.pad,
        py: space.gap,
        bgcolor: 'var(--mortar-paper-78)',
        borderRadius: '6px',
      }}
    >
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography noWrap={true} sx={{ fontSize: 'medium.fontSize', fontWeight: 600 }}>
          {when}
        </Typography>
        <Typography
          noWrap={true}
          title={backup.name}
          sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}
        >
          {meta}
        </Typography>
      </Box>
      <TipIconButton
        label={backup.pinned ? t`Kept forever: allow cleanup` : t`Keep forever`}
        aria-pressed={backup.pinned === true}
        onClick={onPin}
      >
        <Pin size={16} fill={backup.pinned ? 'currentColor' : 'none'} />
      </TipIconButton>
      <TipIconButton
        label={busyGame ? t`Stop the game to restore saves.` : t`Restore ${{ name: when }}`}
        disabled={busyGame || pending}
        // A backup of one save restores it at once; one of several asks which.
        {...((backup.saves ?? []).length > 1
          ? { 'aria-haspopup': 'menu' as const, 'aria-expanded': menuOpen }
          : {})}
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
  const items = useSaveBackups((s) => s.items)
  const status = useSaveBackups((s) => s.status)
  const error = useSaveBackups((s) => s.error)
  const load = useSaveBackups((s) => s.load)
  const restore = useSaveBackups((s) => s.restore)
  const setPinned = useSaveBackups((s) => s.setPinned)
  const openFolder = useSaveBackups((s) => s.openFolder)
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
      <AppDialog
        open={open}
        onClose={onClose}
        title={t`Save backups`}
        size="sm"
        footer={
          <DialogFooter>
            <Button onClick={onClose}>{t`Close`}</Button>
            <Button
              startIcon={<FolderOpen size={16} />}
              onClick={() => {
                openFolder().catch(reportUnexpected)
              }}
            >
              {t`Open backups folder`}
            </Button>
          </DialogFooter>
        }
      >
        {status === 'error' ? (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <Typography sx={{ fontSize: 'base.fontSize', color: 'error.main' }}>
              {t`Could not list backups: ${error}`}
            </Typography>
            <Button size="small" onClick={() => load(game, profile).catch(reportUnexpected)}>
              {t`Retry`}
            </Button>
          </Box>
        ) : null}
        {status === 'loading' && items.length === 0 ? (
          <SkeletonRows label={t`Reading backups…`} count={4} height={36} />
        ) : null}
        {status === 'ready' && items.length === 0 ? (
          <EmptyState
            compact={true}
            icon={<FolderOpen size={28} />}
            title={t`No save backups yet.`}
          >{t`Mortar backs up your saves before playing and before updates. Use Back up now on a save to make one yourself.`}</EmptyState>
        ) : null}
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
          {items.map((b) => (
            <BackupRow
              key={b.name}
              backup={b}
              busyGame={busyGame}
              pending={pending}
              profiles={profiles}
              menuOpen={menu?.backup.name === b.name}
              onRestore={(el) => {
                if ((b.saves ?? []).length > 1) {
                  setMenu({ backup: b, el })
                } else {
                  setConfirm({ backup: b, snaps: b.saves ?? [] })
                }
              }}
              onPin={() => setPinned(b.name, !b.pinned).catch(reportUnexpected)}
            />
          ))}
        </Box>
      </AppDialog>
      <Menu
        open={menu !== null}
        anchorEl={menu?.el}
        onClose={() => {
          setMenu(null)
        }}
      >
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
            label={t`Restore ${{ name: saveName(snap) }}`}
            onClick={() => {
              if (menu) {
                setConfirm({ backup: menu.backup, snaps: [snap] })
              }
              setMenu(null)
            }}
          />
        ))}
      </Menu>
      <ConfirmationDialog
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
        <Typography sx={{ fontSize: 'medium.fontSize' }}>
          {overwriteMessage(confirm?.snaps ?? [], new Set((fits ?? []).map((f) => f.folder)))}
        </Typography>
      </ConfirmationDialog>
    </>
  )
}
