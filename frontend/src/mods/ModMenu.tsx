import { useLingui } from '@lingui/react/macro'
import {
  Divider,
  IconButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Tooltip,
} from '@mui/material'
import {
  Ban,
  Ellipsis,
  ExternalLink,
  Eye,
  FileJson,
  FolderOpen,
  FolderTree,
  Info,
  PackagePlus,
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
  CopyMods,
  FomodPreview,
  ModsDir,
  OpenConsolePath,
  RemoveEntry,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { AddToBundleDialog } from '../bundles/dialogs.tsx'
import { useFomod } from '../fomod/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { SetCategoryDialog } from './CategoryEditor.tsx'
import { useDetail } from './detail.ts'
import { entryOf, modId, nexusIdOf, updateFor } from './lookup.ts'
import { type MenuAnchor, openPage, useContextMenu, useMenuState } from './menu.ts'
import { type ModAction, modActions } from './modActions.ts'
import { useNexusDetails } from './nexusDetails.ts'
import { OtherProfilesDialog } from './OtherProfilesDialog.tsx'
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

function RemoveOtherMenuItem({
  locked,
  close,
  onClick,
}: {
  locked: boolean
  close: () => void
  onClick: () => void
}) {
  const { t } = useLingui()
  return (
    <MenuItem
      disabled={locked}
      onClick={() => {
        close()
        onClick()
      }}
    >
      <ListItemIcon sx={{ color: 'inherit' }}>
        <Trash2 size={ICON_SIZE} />
      </ListItemIcon>
      <ListItemText>{t`Remove from other profiles…`}</ListItemText>
    </MenuItem>
  )
}

function ModActionItems({
  actions,
  items,
  close,
  locked,
  hasFomod,
  mod,
  profile,
  onSetCategory,
  onAlsoAdd,
  onAddBundle,
  labels,
}: {
  actions: ModAction[]
  items: Record<ModAction | 'reinstall', { label: string; icon: ReactNode; run: () => void }>
  close: () => void
  locked: boolean
  hasFomod: boolean
  mod: Mod
  profile: Profile | undefined
  onSetCategory: () => void
  onAlsoAdd: () => void
  onAddBundle: () => void
  labels: { manifest: string; category: string; alsoAdd: string; addBundle: string }
}) {
  return actions.flatMap((a) => [
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
        <ListItemText>{labels.manifest}</ListItemText>
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
        <ListItemText>{labels.category}</ListItemText>
      </MenuItem>
    ) : null,
    a === 'files' ? (
      <MenuItem
        key="also-add"
        disabled={locked}
        onClick={() => {
          close()
          onAlsoAdd()
        }}
      >
        <ListItemText>{labels.alsoAdd}</ListItemText>
      </MenuItem>
    ) : null,
    a === 'files' ? (
      <MenuItem
        key="add-bundle"
        disabled={locked}
        onClick={() => {
          close()
          onAddBundle()
        }}
      >
        <ListItemIcon sx={{ color: 'inherit' }}>
          <PackagePlus size={ICON_SIZE} />
        </ListItemIcon>
        <ListItemText>{labels.addBundle}</ListItemText>
      </MenuItem>
    ) : null,
  ])
}

function ModMenuItems({
  mod,
  close,
  onSetCategory,
  onAlsoAdd,
  onAddBundle,
  onRemoveOther,
}: {
  mod: Mod
  close: () => void
  onSetCategory: () => void
  onAlsoAdd: () => void
  onAddBundle: () => void
  onRemoveOther: () => void
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
  return [
    ...ModActionItems({
      actions: modActions(state),
      items,
      close,
      locked,
      hasFomod,
      mod,
      profile,
      onSetCategory,
      onAlsoAdd,
      onAddBundle,
      labels: {
        manifest: t`Open manifest.json`,
        category: t`Set category…`,
        alsoAdd: t`Also add to…`,
        addBundle: t`Add to bundle…`,
      },
    }),
    <RemoveOtherMenuItem
      key="remove-other"
      locked={locked}
      close={close}
      onClick={onRemoveOther}
    />,
  ]
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
  const { t } = useLingui()
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === s.openId))
  const byId = useNexusDetails((s) => s.byId)
  const entry = (profile?.entries ?? []).find((e) => e.key === mod.key)
  const nexusCategory =
    profile === undefined ? '' : (byId[nexusIdOf(profile, mod)]?.details?.category ?? '')
  const [categoryOpen, setCategoryOpen] = useState(false)
  const [alsoOpen, setAlsoOpen] = useState(false)
  const [bundleOpen, setBundleOpen] = useState(false)
  const [removeOtherOpen, setRemoveOtherOpen] = useState(false)
  const game = useProfiles((s) => s.game?.id ?? '')
  const currentProfileId = useProfiles((s) => s.openId)
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
        <ModMenuItems
          mod={mod}
          close={onClose}
          onSetCategory={() => setCategoryOpen(true)}
          onAlsoAdd={() => setAlsoOpen(true)}
          onAddBundle={() => setBundleOpen(true)}
          onRemoveOther={() => setRemoveOtherOpen(true)}
        />
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
      <OtherProfilesDialog
        open={alsoOpen}
        onClose={() => setAlsoOpen(false)}
        game={game}
        currentProfileId={currentProfileId}
        uniqueId={mod.uniqueId}
        title={t`Also add ${mod.name} to…`}
        confirmLabel={t`Add`}
        onConfirm={async (profiles) => {
          await Promise.all(
            profiles.map((other) => CopyMods(game, currentProfileId, other.id, [mod.uniqueId])),
          )
        }}
      />
      <OtherProfilesDialog
        open={removeOtherOpen}
        onClose={() => setRemoveOtherOpen(false)}
        game={game}
        currentProfileId={currentProfileId}
        uniqueId={mod.uniqueId}
        mode="remove"
        title={t`Remove ${mod.name} from other profiles`}
        confirmLabel={t`Remove`}
        onConfirm={async (profiles) => {
          await Promise.all(profiles.map((other) => RemoveEntry(game, other.id, mod.key)))
          useToasts.getState().push({ kind: 'success', title: t`Mods removed from other profiles` })
        }}
      />
      <AddToBundleDialog
        open={bundleOpen}
        onClose={() => setBundleOpen(false)}
        game={game}
        profileId={currentProfileId}
        uniqueIds={[mod.uniqueId]}
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
      <Tooltip title={t`More actions for ${mod.name}`}>
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
      </Tooltip>
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
