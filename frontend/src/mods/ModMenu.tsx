import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Menu } from '@mui/material'
import {
  ArrowDown,
  ArrowUp,
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
  Settings2,
  Trash2,
} from 'lucide-react'
import { type ReactNode, useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  CopyMods,
  FomodPreview,
  MovePackage,
  RemoveEntry,
  SetModEnabled,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { AddToBundleDialog } from '../bundles/dialogs.tsx'
import { useFomod } from '../fomod/store.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { IconAction } from '../shell/IconAction.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { AddToGroupDialog } from './AddToGroupDialog.tsx'
import { actingMods, toggleActing } from './actingMods.ts'
import { SetCategoryDialog } from './CategoryEditor.tsx'
import { useDetail } from './detail.ts'
import { EverywhereMenuItem } from './EverywhereMenuItem.tsx'
import { ModGroupItems } from './GroupMenu.tsx'
import { modId, nexusIdOf, updateFor } from './lookup.ts'
import { ModActionItems } from './ModActionItems.tsx'
import {
  extraFileArchive,
  extraFileLabel,
  ICON_SIZE,
  type MenuAnchor,
  openPage,
  samePageSiblings,
  useContextMenu,
  useMenuState,
} from './menu.ts'
import { type ModAction, modActions, openPageLabel } from './modActions.ts'
import { siteCategory } from './modSearch.ts'
import { useNexusDetails } from './nexusDetails.ts'
import { OtherProfilesDialog } from './OtherProfilesDialog.tsx'
import { SplitCombineItems } from './SplitCombineItems.tsx'
import { useSelection } from './selection.ts'
import { useMods } from './store.ts'
import { hasConfig, openConfigPage, useConfigList } from './typedConfig/configList.ts'
import { useUpdates } from './updates.ts'
import { useLocked } from './useLocked.ts'

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

function ModMenuItems({
  mod,
  close,
  onSetCategory,
  onAlsoAdd,
  onAddBundle,
  onRemoveOther,
  trailing,
}: {
  mod: Mod
  close: () => void
  onSetCategory: () => void
  onAlsoAdd: () => void
  onAddBundle: () => void
  onRemoveOther: () => void
  trailing: ReactNode[]
}) {
  const { t, i18n } = useLingui()
  const showFiles = useMods((s) => s.showFiles)
  const askRemove = useMods((s) => s.askRemove)
  const selectedIds = useSelection((s) => s.ids)
  const acting = selectedIds.length > 1 ? actingMods(mod) : [mod]
  const setPinned = useMods((s) => s.setPinned)
  const setSkipVersion = useMods((s) => s.setSkipVersion)
  const show = useDetail((s) => s.show)
  const page = useMods((s) => s.pages[modId(mod)])
  const state = useMenuState(mod)
  const locked = useLocked()
  const profile = useProfiles(openProfileOf)
  const game = useProfiles((s) => s.game)
  const update = useUpdates((s) => updateFor(s.updates, mod, profile))
  const currentEntry = (profile?.entries ?? []).find((e) => e.key === mod.key)
  const hasFomod = Boolean(currentEntry?.fomod && Object.keys(currentEntry.fomod).length > 0)
  const extras = currentEntry?.extraStoreKeys ?? []
  const siblings = samePageSiblings(profile, currentEntry)
  const files = useNexusDetails((s) => s.byId[currentEntry?.source?.modId ?? 0]?.details?.files)
  const configured = useConfigList((st) => hasConfig(st.byProfile[profile?.id ?? '']?.list, mod.id))
  const items: Record<
    ModAction | 'reinstall',
    { label: string; icon: ReactNode; run: () => void }
  > = {
    toggle: {
      label: mod.enabled ? t`Disable` : t`Enable`,
      icon: mod.enabled ? <PowerOff size={ICON_SIZE} /> : <Power size={ICON_SIZE} />,
      run: () => toggleActing(mod).catch(reportUnexpected),
    },
    details: {
      label: t`More details`,
      icon: <Info size={ICON_SIZE} />,
      run: () => show(mod),
    },
    page: {
      label: openPageLabel(i18n, state.host),
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
        if (!(game && profile && currentEntry)) {
          return
        }
        openFomodReinstall(game.id, profile, currentEntry.key)
      },
    },
    pin: {
      label: state.pinned ? t`Unpin version` : t`Pin this version`,
      icon: state.pinned ? <PinOff size={ICON_SIZE} /> : <Pin size={ICON_SIZE} />,
      run: () => setPinned(mod, !state.pinned).catch(reportUnexpected),
    },
    skip: {
      label: update ? t`Skip this update` : t`Show skipped update`,
      icon: update ? <Ban size={ICON_SIZE} /> : <Eye size={ICON_SIZE} />,
      run: () => setSkipVersion(mod, update ? update.version : '').catch(reportUnexpected),
    },
    remove: {
      label:
        acting.length > 1
          ? t`${plural(acting.length, { one: 'Remove # mod', other: 'Remove # mods' })}`
          : t`Remove`,
      icon: <Trash2 size={ICON_SIZE} />,
      run: () => askRemove(acting),
    },
  }
  const splitCombine = (
    <SplitCombineItems
      extras={extras}
      siblings={siblings}
      locked={locked}
      close={close}
      gameId={game?.id ?? ''}
      profileId={profile?.id ?? ''}
      entryKey={currentEntry?.key ?? ''}
      extraLabel={(extraKey) => extraFileLabel(currentEntry, extraKey, files)}
      extraArchive={(extraKey) => extraFileArchive(extraKey, files)}
    />
  )
  return [
    ...ModActionItems({
      actions: modActions(state),
      items,
      close,
      locked,
      hasFomod,
      onSetCategory,
      onAlsoAdd,
      onAddBundle,
      onRemoveOther,
      onEditConfig: configured ? () => openConfigPage(mod.id) : undefined,
      labels: {
        editConfig: t`Edit config`,
        category: t`Set category…`,
        alsoAdd: t`Also add to…`,
        addBundle: t`Add to bundle…`,
      },
      splitCombine,
      trailing,
    }),
  ]
}

// A later package wins a file two mods both provide, so moving one down makes it win more.
function PackageMoveItems({ mod, close }: { mod: Mod; close: () => void }) {
  const { t } = useLingui()
  const locked = useLocked()
  const game = useProfiles((s) => s.game)
  const profile = useProfiles(openProfileOf)
  if (game?.deploy !== 'profile' || !profile) {
    return null
  }
  const packages = (profile.entries ?? []).filter((e) => !e.overlayOf)
  const place = packages.findIndex((e) => e.key === mod.key)
  const move = (delta: -1 | 1) => {
    close()
    MovePackage(game.id, profile.id, mod.key, delta)
      .then((p) => useProfiles.getState().replace(p))
      .catch(reportUnexpected)
  }
  return [
    <MenuAction
      key="package-up"
      disabled={locked || place <= 0}
      icon={<ArrowUp size={ICON_SIZE} />}
      label={t`Move up (loses files)`}
      onClick={() => move(-1)}
    />,
    <MenuAction
      key="package-down"
      disabled={locked || place < 0 || place === packages.length - 1}
      icon={<ArrowDown size={ICON_SIZE} />}
      label={t`Move down (wins files)`}
      onClick={() => move(1)}
    />,
  ]
}

function RemoveFromOthersDialog({
  open,
  onClose,
  mod,
  game,
  currentProfileId,
}: {
  open: boolean
  onClose: () => void
  mod: Mod
  game: string
  currentProfileId: string
}) {
  const { t } = useLingui()
  return (
    <OtherProfilesDialog
      open={open}
      onClose={onClose}
      game={game}
      currentProfileId={currentProfileId}
      id={mod.id}
      mode="remove"
      title={t`Remove ${mod.name} from other profiles`}
      helper={t`Where it came in one download with other mods, it is disabled instead.`}
      confirmLabel={t`Remove`}
      onConfirm={async (profiles, _pinned, rows) => {
        const changed = await Promise.all(
          profiles.map((other) => {
            const row = rows.find((candidate) => candidate.profileId === other.id)
            const entryMods =
              useProfiles
                .getState()
                .profiles.find((candidate) => candidate.id === other.id)
                ?.entries?.find((profileEntry) => profileEntry.key === row?.key)?.mods ?? []
            return entryMods.length > 1
              ? SetModEnabled(game, other.id, row?.key ?? '', mod.id, false).then((result) => {
                  useProfiles.getState().replace(result.profile)
                  return result.profile
                })
              : RemoveEntry(game, other.id, row?.key ?? '')
          }),
        )
        useToasts.getState().push({
          kind: 'success',
          title: t`${mod.name} removed from ${plural(profiles.length, { one: '# profile', other: '# profiles' })}`,
          changes: changed.map((p) => p.lastChange ?? ''),
        })
      }}
    />
  )
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
  const profile = useProfiles(openProfileOf)
  const byId = useNexusDetails((s) => s.byId)
  const entry = (profile?.entries ?? []).find((e) => e.key === mod.key)
  const nexusCategory =
    profile === undefined
      ? ''
      : (siteCategory(entry, byId[nexusIdOf(profile, mod)]?.details?.category) ?? '')
  const [categoryOpen, setCategoryOpen] = useState(false)
  const [alsoOpen, setAlsoOpen] = useState(false)
  const [bundleOpen, setBundleOpen] = useState(false)
  const [groupOpen, setGroupOpen] = useState(false)
  const [removeOtherOpen, setRemoveOtherOpen] = useState(false)
  const game = useProfiles((s) => s.game?.id ?? '')
  const currentProfileId = useProfiles((s) => s.openId)
  const update = useUpdates((s) => updateFor(s.updates, mod, profile))
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
          trailing={[
            <PackageMoveItems key="package-move" mod={mod} close={onClose} />,
            <ModGroupItems
              key="groups"
              profile={profile}
              entryKey={mod.key}
              close={onClose}
              onAdd={() => setGroupOpen(true)}
            />,
            update ? (
              <EverywhereMenuItem key="everywhere" game={game} id={mod.id} close={onClose} />
            ) : null,
          ]}
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
        id={mod.id}
        title={t`Also add ${mod.name} to…`}
        confirmLabel={t`Add`}
        onConfirm={async (profiles) => {
          await Promise.all(
            profiles.map((other) => CopyMods(game, currentProfileId, other.id, [mod.id])),
          )
        }}
      />
      <RemoveFromOthersDialog
        open={removeOtherOpen}
        onClose={() => setRemoveOtherOpen(false)}
        mod={mod}
        game={game}
        currentProfileId={currentProfileId}
      />
      <AddToBundleDialog
        open={bundleOpen}
        onClose={() => setBundleOpen(false)}
        game={game}
        profileId={currentProfileId}
        ids={[mod.id]}
      />
      <AddToGroupDialog
        open={groupOpen}
        onClose={() => setGroupOpen(false)}
        entryKey={mod.key}
        profile={profile}
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
      <IconAction
        label={t`More actions for ${mod.name}`}
        icon={<Ellipsis size={18} />}
        menu={true}
        onClick={(e) => {
          e.stopPropagation()
          setAnchor(e.currentTarget)
        }}
      />
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
