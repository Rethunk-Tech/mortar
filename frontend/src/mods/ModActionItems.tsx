import { useLingui } from '@lingui/react/macro'
import { Divider, ListItemIcon, ListItemText, MenuItem } from '@mui/material'
import { CopyPlus, FileJson, FolderTree, PackagePlus, Trash2 } from 'lucide-react'
import type { ReactNode } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  ModsDir,
  OpenConsolePath,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
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

export function ModActionItems({
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
}) {
  const renderAction = (action: ModAction | 'reinstall', disabled = false) => (
    <MenuItem
      key={action}
      disabled={disabled}
      onClick={() => {
        close()
        items[action].run()
      }}
    >
      <ListItemIcon sx={{ color: 'inherit' }}>{items[action].icon}</ListItemIcon>
      <ListItemText>{items[action].label}</ListItemText>
    </MenuItem>
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
      </MenuItem>,
    )
    result.push(<Divider key="organisation-divider" />)
    result.push(
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
      </MenuItem>,
    )
    result.push(
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
      </MenuItem>,
    )
    if (has('pin')) {
      result.push(renderAction('pin'))
    }
    if (has('skip')) {
      result.push(renderAction('skip'))
    }
    result.push(<Divider key="other-profiles-divider" />)
    result.push(
      <MenuItem
        key="also-add"
        disabled={locked}
        onClick={() => {
          close()
          onAlsoAdd()
        }}
      >
        <ListItemIcon sx={{ color: 'inherit' }}>
          <CopyPlus size={ICON_SIZE} />
        </ListItemIcon>
        <ListItemText>{labels.alsoAdd}</ListItemText>
      </MenuItem>,
    )
  }
  result.push(
    <RemoveOtherMenuItem
      key="remove-other"
      locked={locked}
      close={close}
      onClick={onRemoveOther}
    />,
  )
  if (splitCombine) {
    result.push(splitCombine)
  }
  if (has('remove')) {
    result.push(<Divider key="remove-divider" />)
    result.push(
      <MenuItem
        key="remove"
        disabled={locked}
        sx={{ color: 'error.main' }}
        onClick={() => {
          close()
          items.remove.run()
        }}
      >
        <ListItemIcon sx={{ color: 'inherit' }}>{items.remove.icon}</ListItemIcon>
        <ListItemText>{items.remove.label}</ListItemText>
      </MenuItem>,
    )
  }
  return result
}
