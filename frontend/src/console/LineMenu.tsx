import { useLingui } from '@lingui/react/macro'
import { ListItemIcon, ListItemText, Menu, MenuItem } from '@mui/material'
import { Copy, EyeOff } from 'lucide-react'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { HideLogSource } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { copyText } from '../share/copyText.ts'
import { reportError } from '../toasts/report.ts'
import { sourceOf } from './logSource.ts'

/** A Console line's right-click menu: copy its message (with its further lines), or hide the lines its source writes,
 * about the object it names or about every object. */
export function LineMenu({
  game,
  at,
  group,
  onClose,
}: {
  game: string
  at: { top: number; left: number } | null
  group: readonly Entry[]
  onClose: () => void
}) {
  const { t } = useLingui()
  const source = sourceOf(group)
  const hide = (about: string) => {
    if (source) {
      HideLogSource(game, { ...source, object: about }).catch(
        reportError(t`Could not hide that message`),
      )
    }
    onClose()
  }
  const object = source?.object ?? ''
  const name = object.split('/').at(-1) ?? ''
  return (
    <Menu
      open={at !== null}
      onClose={onClose}
      anchorReference="anchorPosition"
      anchorPosition={at ?? undefined}
    >
      <MenuItem
        onClick={() => {
          copyText(group.map((e) => e.message).join('\n'), t`Message copied`)
          onClose()
        }}
      >
        <ListItemIcon>
          <Copy size={16} aria-hidden={true} />
        </ListItemIcon>
        <ListItemText>{t`Copy this message`}</ListItemText>
      </MenuItem>
      {object ? (
        <MenuItem title={object} onClick={() => hide(object)}>
          <ListItemIcon>
            <EyeOff size={16} aria-hidden={true} />
          </ListItemIcon>
          <ListItemText>{t`Hide this message from ${name}`}</ListItemText>
        </MenuItem>
      ) : null}
      {source ? (
        <MenuItem onClick={() => hide('')}>
          <ListItemIcon>
            <EyeOff size={16} aria-hidden={true} />
          </ListItemIcon>
          <ListItemText>
            {object ? t`Hide this message from every object` : t`Hide this message`}
          </ListItemText>
        </MenuItem>
      ) : null}
    </Menu>
  )
}
