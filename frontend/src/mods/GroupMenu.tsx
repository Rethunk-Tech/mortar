import { useLingui } from '@lingui/react/macro'
import { Menu } from '@mui/material'
import { FolderMinus, MoreHorizontal, Pencil, Power, PowerOff, Trash2 } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { PromptDialog } from '../shell/PromptDialog.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { AddToGroupMenuItem } from './AddToGroupDialog.tsx'
import { LockedReason } from './LockedReason.tsx'
import { useMods } from './store.ts'
import { deleteGroup, removeModFromGroup, renameGroup, setGroupEnabled } from './storeEntries.ts'
import { useLocked } from './useLocked.ts'

const ICON_SIZE = 16
const MAX_NAME = 60

// The group header's ⋯ menu: rename, delete (its mods stay in the profile) and switch every mod of the group.
export function GroupMenu({ name }: { name: string }) {
  const { t } = useLingui()
  const locked = useLocked()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [renaming, setRenaming] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const close = () => setAnchor(null)
  const switchAll = (on: boolean) =>
    setGroupEnabled(name, on)
      .then(() => useMods.getState().load())
      .catch(reportUnexpected)
  const item = (label: string, icon: ReactNode, run: () => void) => (
    <MenuAction
      disabled={locked}
      icon={icon}
      label={label}
      onClick={() => {
        close()
        run()
      }}
    />
  )
  return (
    <>
      <TipIconButton label={t`Group actions`} onClick={(e) => setAnchor(e.currentTarget)}>
        <MoreHorizontal size={ICON_SIZE} />
      </TipIconButton>
      <Menu open={anchor !== null} anchorEl={anchor} onClose={close}>
        <LockedReason locked={locked}>
          {item(t`Rename…`, <Pencil size={ICON_SIZE} />, () => setRenaming(true))}
        </LockedReason>
        <LockedReason locked={locked}>
          {item(t`Enable all`, <Power size={ICON_SIZE} />, () => switchAll(true))}
        </LockedReason>
        <LockedReason locked={locked}>
          {item(t`Disable all`, <PowerOff size={ICON_SIZE} />, () => switchAll(false))}
        </LockedReason>
        <LockedReason locked={locked}>
          {item(t`Delete group (mods stay)`, <Trash2 size={ICON_SIZE} />, () => setDeleting(true))}
        </LockedReason>
      </Menu>
      <PromptDialog
        open={renaming}
        title={t`Rename group`}
        label={t`Group name`}
        initial={name}
        maxLength={MAX_NAME}
        confirmLabel={t`Rename`}
        onCancel={() => setRenaming(false)}
        onSubmit={(next) => {
          setRenaming(false)
          renameGroup(name, next).catch(reportUnexpected)
        }}
      />
      <ConfirmDialog
        open={deleting}
        color="error"
        title={t`Delete group ${name}?`}
        body={t`The mods in it stay in the profile; only the group is deleted.`}
        confirmLabel={t`Delete group`}
        onCancel={() => setDeleting(false)}
        onConfirm={() => {
          setDeleting(false)
          deleteGroup(name).catch(reportUnexpected)
        }}
      />
    </>
  )
}

// The mod menu's group items: add to a group, and one "Remove from <group>" for each group that holds the entry.
export function ModGroupItems({
  profile,
  entryKey,
  close,
  onAdd,
}: {
  profile: Profile | undefined
  entryKey: string
  close: () => void
  onAdd: () => void
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const holding = (profile?.groups ?? []).filter((g) => (g.keys ?? []).includes(entryKey))
  return (
    <>
      <AddToGroupMenuItem
        locked={locked}
        onClick={() => {
          close()
          onAdd()
        }}
      />
      {holding.map((g) => (
        <LockedReason key={g.name} locked={locked}>
          <MenuAction
            disabled={locked}
            icon={<FolderMinus size={ICON_SIZE} />}
            label={t`Remove from ${g.name}`}
            onClick={() => {
              close()
              removeModFromGroup(entryKey, g.name ?? '').catch(reportUnexpected)
            }}
          />
        </LockedReason>
      ))}
    </>
  )
}
