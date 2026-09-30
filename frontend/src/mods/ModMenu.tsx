import { useLingui } from '@lingui/react/macro'
import { Divider, IconButton, ListItemIcon, ListItemText, Menu, MenuItem } from '@mui/material'
import {
  Ban,
  Ellipsis,
  ExternalLink,
  Eye,
  FolderOpen,
  Info,
  Pin,
  PinOff,
  Power,
  PowerOff,
  Trash2,
} from 'lucide-react'
import { type ReactNode, useState } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useDetail } from './detail.ts'
import { modId, updateFor } from './lookup.ts'
import { type MenuAnchor, openPage, useContextMenu, useMenuState } from './menu.ts'
import { type ModAction, modActions } from './modActions.ts'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'
import { useLocked } from './useLocked.ts'

const ICON_SIZE = 16

function ModMenuItems({ mod, close }: { mod: Mod; close: () => void }) {
  const { t } = useLingui()
  const showFiles = useMods((s) => s.showFiles)
  const askRemove = useMods((s) => s.askRemove)
  const setEnabled = useMods((s) => s.setEnabled)
  const setPinned = useMods((s) => s.setPinned)
  const setSkipVersion = useMods((s) => s.setSkipVersion)
  const show = useDetail((s) => s.show)
  const setOpen = useDetail((s) => s.setOpen)
  const page = useMods((s) => s.pages[modId(mod)])
  const state = useMenuState(mod)
  const locked = useLocked()
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === s.openId))
  const update = useUpdates((s) => updateFor(s.updates, mod, profile))
  const items: Record<ModAction, { label: string; icon: ReactNode; run: () => void }> = {
    toggle: {
      label: mod.enabled ? t`Disable` : t`Enable`,
      icon: mod.enabled ? <PowerOff size={ICON_SIZE} /> : <Power size={ICON_SIZE} />,
      run: () => setEnabled(mod, !mod.enabled).catch(reportUnexpected),
    },
    details: {
      label: t`More details`,
      icon: <Info size={ICON_SIZE} />,
      run: () => {
        show(mod)
        setOpen(true)
      },
    },
    page: {
      label: state.host === 'github' ? t`Open on GitHub` : t`Open on Nexus`,
      icon: <ExternalLink size={ICON_SIZE} />,
      run: () => {
        if (page) {
          openPage(page).catch(reportUnexpected)
        }
      },
    },
    files: {
      label: t`Show files`,
      icon: <FolderOpen size={ICON_SIZE} />,
      run: () => showFiles(mod).catch(reportUnexpected),
    },
    pin: {
      label: state.pinned ? t`Unpin` : t`Pin this version`,
      icon: state.pinned ? <PinOff size={ICON_SIZE} /> : <Pin size={ICON_SIZE} />,
      run: () => setPinned(mod, !state.pinned).catch(reportUnexpected),
    },
    skip: {
      label: update ? t`Skip this update` : t`Show skipped update`,
      icon: update ? <Ban size={ICON_SIZE} /> : <Eye size={ICON_SIZE} />,
      run: () => setSkipVersion(mod, update ? update.version : '').catch(reportUnexpected),
    },
    remove: {
      label: t`Remove`,
      icon: <Trash2 size={ICON_SIZE} />,
      run: () => askRemove(mod),
    },
  }
  return modActions(state).flatMap((a) => [
    a === 'remove' ? <Divider key="divider" /> : null,
    <MenuItem
      key={a}
      disabled={locked && (a === 'toggle' || a === 'remove')}
      sx={a === 'remove' ? { color: 'error.main' } : undefined}
      onClick={() => {
        close()
        items[a].run()
      }}
    >
      <ListItemIcon sx={{ color: 'inherit' }}>{items[a].icon}</ListItemIcon>
      <ListItemText>{items[a].label}</ListItemText>
    </MenuItem>,
  ])
}

function ModActionMenu({
  mod,
  anchor,
  onClose,
}: {
  mod: Mod
  anchor: MenuAnchor
  onClose: () => void
}) {
  const position =
    'el' in anchor ? {} : { anchorReference: 'anchorPosition' as const, anchorPosition: anchor }
  return (
    <Menu
      open={true}
      onClose={onClose}
      anchorEl={'el' in anchor ? anchor.el : undefined}
      {...position}
    >
      <ModMenuItems mod={mod} close={onClose} />
    </Menu>
  )
}

// The ⋯ button of a card.
export function ModMenu({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  return (
    <>
      <IconButton
        aria-label={t`More actions for ${mod.name}`}
        size="small"
        onClick={(e) => {
          e.stopPropagation()
          setAnchor(e.currentTarget)
        }}
      >
        <Ellipsis size={18} />
      </IconButton>
      {anchor ? (
        <ModActionMenu mod={mod} anchor={{ el: anchor }} onClose={() => setAnchor(null)} />
      ) : null}
    </>
  )
}

// The right-click menu, shared by every card and row of the mods screen.
export function ModContextMenu() {
  const target = useContextMenu((s) => s.target)
  const close = useContextMenu((s) => s.close)
  if (!target) {
    return null
  }
  return <ModActionMenu {...target} onClose={close} />
}
