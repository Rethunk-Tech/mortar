import { useLingui } from '@lingui/react/macro'
import { Divider, IconButton, ListItemIcon, ListItemText, Menu, MenuItem } from '@mui/material'
import {
  Ban,
  Ellipsis,
  ExternalLink,
  Eye,
  FileJson,
  FolderOpen,
  FolderTree,
  Info,
  Pin,
  PinOff,
  Power,
  PowerOff,
  Settings2,
  Trash2,
} from 'lucide-react'
import { type ReactNode, useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  FomodPreview,
  ModsDir,
  OpenConsolePath,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useFomod } from '../fomod/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { SetCategoryDialog } from './CategoryEditor.tsx'
import { useDetail } from './detail.ts'
import { entryOf, modId, nexusIdOf, updateFor } from './lookup.ts'
import { type MenuAnchor, openPage, useContextMenu, useMenuState } from './menu.ts'
import { type ModAction, modActions } from './modActions.ts'
import { useNexusDetails } from './nexusDetails.ts'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'
import { useLocked } from './useLocked.ts'

const ICON_SIZE = 16
const TRAILING_SEP = /[/\\]+$/

function openFomodReinstall(gameId: string, profile: Profile, key: string) {
  const entry = (profile.entries ?? []).find((e) => e.key === key)
  if (!entry) {
    return
  }
  FomodPreview(gameId, profile.id, entry.key, entry.fomod ?? {})
    .then((ask) =>
      useFomod.getState().open({
        game: gameId,
        profileId: profile.id,
        key: entry.key,
        source: entry.source,
        ask,
      }),
    )
    .catch(reportUnexpected)
}

function openManifestOf(mod: Mod, profile: Profile | undefined) {
  const { game: currentGame, openId } = useProfiles.getState()
  const gameId = currentGame?.id
  if (!(gameId && openId)) {
    return
  }
  const folder =
    (entryOf(profile, mod.key)?.mods ?? []).find((m) => m.uniqueId === mod.uniqueId)?.folder ?? '.'
  const nested = folder !== '' && folder !== '.'
  const rel = nested ? `${folder}/manifest.json` : 'manifest.json'
  ModsDir(gameId, openId)
    .then((dir) =>
      OpenConsolePath(gameId, openId, `${dir.replace(TRAILING_SEP, '')}/${mod.key}/${rel}`),
    )
    .catch(reportUnexpected)
}

function ModMenuItems({
  mod,
  close,
  onSetCategory,
}: {
  mod: Mod
  close: () => void
  onSetCategory: () => void
}) {
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
  const game = useProfiles((s) => s.game)
  const update = useUpdates((s) => updateFor(s.updates, mod, profile))
  const entry = (profile?.entries ?? []).find((e) => e.key === mod.key)
  const hasFomod = Boolean(entry?.fomod && Object.keys(entry.fomod).length > 0)
  const items: Record<
    ModAction | 'reinstall',
    { label: string; icon: ReactNode; run: () => void }
  > = {
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
    reinstall: {
      label: t`Reinstall with options…`,
      icon: <Settings2 size={ICON_SIZE} />,
      run: () => {
        if (!(game && profile && entry)) {
          return
        }
        openFomodReinstall(game.id, profile, entry.key)
      },
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
    a === 'files' && hasFomod ? (
      <MenuItem
        key="reinstall"
        disabled={locked}
        onClick={() => {
          close()
          items.reinstall.run()
        }}
      >
        <ListItemIcon sx={{ color: 'inherit' }}>{items.reinstall.icon}</ListItemIcon>
        <ListItemText>{items.reinstall.label}</ListItemText>
      </MenuItem>
    ) : null,
    a === 'files' ? (
      <MenuItem
        key="manifest"
        onClick={() => {
          close()
          openManifestOf(mod, profile)
        }}
      >
        <ListItemIcon sx={{ color: 'inherit' }}>
          <FileJson size={ICON_SIZE} />
        </ListItemIcon>
        <ListItemText>{t`Open manifest.json`}</ListItemText>
      </MenuItem>
    ) : null,
    a === 'files' ? (
      <MenuItem
        key="category"
        onClick={() => {
          close()
          onSetCategory()
        }}
      >
        <ListItemIcon sx={{ color: 'inherit' }}>
          <FolderTree size={ICON_SIZE} />
        </ListItemIcon>
        <ListItemText>{t`Set category…`}</ListItemText>
      </MenuItem>
    ) : null,
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
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === s.openId))
  const byId = useNexusDetails((s) => s.byId)
  const entry = (profile?.entries ?? []).find((e) => e.key === mod.key)
  const nexusCategory =
    profile === undefined ? '' : (byId[nexusIdOf(profile, mod)]?.details?.category ?? '')
  const [categoryOpen, setCategoryOpen] = useState(false)
  const position =
    'el' in anchor ? {} : { anchorReference: 'anchorPosition' as const, anchorPosition: anchor }
  return (
    <>
      <Menu
        open={!categoryOpen}
        onClose={onClose}
        anchorEl={'el' in anchor ? anchor.el : undefined}
        {...position}
      >
        <ModMenuItems mod={mod} close={onClose} onSetCategory={() => setCategoryOpen(true)} />
      </Menu>
      <SetCategoryDialog
        open={categoryOpen}
        onClose={() => {
          setCategoryOpen(false)
          onClose()
        }}
        modKey={mod.key}
        nexusCategory={nexusCategory}
        currentOverride={entry?.categoryOverride ?? ''}
      />
    </>
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
