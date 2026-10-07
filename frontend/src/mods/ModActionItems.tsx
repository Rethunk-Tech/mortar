import { useLingui } from '@lingui/react/macro'
import { Divider } from '@mui/material'
import { CopyPlus, FileJson, FolderTree, PackagePlus, Trash2 } from 'lucide-react'
import type { ReactNode } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  ModsDir,
  OpenConsolePath,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { entryOf } from './lookup.ts'
import { ICON_SIZE } from './menu.ts'
import type { ModAction } from './modActions.ts'

const TRAILING_SEP = /[/\\]+$/

function openManifestOf(mod: Mod, profile: Profile | undefined) {
  const { game: currentGame, openId } = useProfiles.getState()
  const gameId = currentGame?.id
  if (!(gameId && openId)) {
    return
  }
  const folder = (entryOf(profile, mod.key)?.mods ?? []).find((m) => m.id === mod.id)?.folder ?? '.'
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
  tooltip,
  close,
  onClick,
}: {
  locked: boolean
  tooltip: string
  close: () => void
  onClick: () => void
}) {
  const { t } = useLingui()
  return (
    <MenuAction
      disabled={locked}
      tooltip={tooltip}
      icon={<Trash2 size={ICON_SIZE} />}
      label={t`Remove from other profiles…`}
      onClick={() => {
        close()
        onClick()
      }}
    />
  )
}

function RemoveMenuItem({
  item,
  locked,
  tooltip,
  close,
}: {
  item: { label: string; icon: ReactNode; run: () => void }
  locked: boolean
  tooltip: string
  close: () => void
}) {
  return (
    <MenuAction
      disabled={locked}
      tone="error"
      icon={item.icon}
      label={item.label}
      tooltip={locked ? tooltip : undefined}
      onClick={() => {
        close()
        item.run()
      }}
    />
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
  onRemoveOther,
  labels,
  splitCombine,
  trailing,
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
  onRemoveOther: () => void
  labels: { manifest: string; category: string; alsoAdd: string; addBundle: string }
  splitCombine: ReactNode
  trailing: ReactNode[]
}) {
  const { t } = useLingui()
  const lockedTip = t`Stop the game to change mods.`
  const closeThen = (fn: () => void) => () => {
    close()
    fn()
  }
  const renderAction = (action: ModAction | 'reinstall', disabled = false) => (
    <MenuAction
      key={action}
      disabled={disabled}
      tooltip={lockedTip}
      icon={items[action].icon}
      label={items[action].label}
      onClick={closeThen(items[action].run)}
    />
  )
  const has = (action: ModAction) => actions.includes(action)
  const result: ReactNode[] = []
  if (has('toggle')) {
    result.push(renderAction('toggle', locked))
  }
  if (has('details')) {
    result.push(renderAction('details'))
  }
  result.push(<Divider key="source-divider" />)
  if (has('page')) {
    result.push(renderAction('page'))
  }
  if (has('files')) {
    result.push(renderAction('files'))
    if (hasFomod) {
      result.push(renderAction('reinstall', locked))
    }
    result.push(
      <MenuAction
        key="manifest"
        icon={<FileJson size={ICON_SIZE} />}
        label={labels.manifest}
        onClick={closeThen(() => openManifestOf(mod, profile))}
      />,
    )
    result.push(<Divider key="organisation-divider" />)
    result.push(
      <MenuAction
        key="category"
        icon={<FolderTree size={ICON_SIZE} />}
        label={labels.category}
        onClick={closeThen(onSetCategory)}
      />,
    )
    result.push(
      <MenuAction
        key="add-bundle"
        disabled={locked}
        tooltip={lockedTip}
        icon={<PackagePlus size={ICON_SIZE} />}
        label={labels.addBundle}
        onClick={closeThen(onAddBundle)}
      />,
    )
    if (has('pin')) {
      result.push(renderAction('pin'))
    }
    if (has('skip')) {
      result.push(renderAction('skip'))
    }
    result.push(<Divider key="other-profiles-divider" />)
    result.push(
      <MenuAction
        key="also-add"
        disabled={locked}
        tooltip={lockedTip}
        icon={<CopyPlus size={ICON_SIZE} />}
        label={labels.alsoAdd}
        onClick={closeThen(onAlsoAdd)}
      />,
    )
  }
  result.push(
    <RemoveOtherMenuItem
      key="remove-other"
      locked={locked}
      tooltip={lockedTip}
      close={close}
      onClick={onRemoveOther}
    />,
  )
  if (splitCombine) {
    result.push(splitCombine)
  }
  result.push(...trailing)
  if (has('remove')) {
    result.push(<Divider key="remove-divider" />)
    result.push(
      <RemoveMenuItem
        key="remove"
        item={items.remove}
        locked={locked}
        tooltip={lockedTip}
        close={close}
      />,
    )
  }
  return result
}

export { ModActionItems }
