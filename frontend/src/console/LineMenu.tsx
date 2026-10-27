import { useLingui } from '@lingui/react/macro'
import { Menu } from '@mui/material'
import { Copy, EyeOff } from 'lucide-react'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { HideLogSource } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { copyText } from '../share/copyText.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportError } from '../toasts/report.ts'
import { formatFull } from './filter.ts'
import { sourceOf } from './logSource.ts'

// Type::Method without the type's namespace, as people read it: PlayerControllerB.GrabObjectClientRpc.
function shortMethod(method: string) {
  const [type = '', name = ''] = method.split('::')
  return `${type.split('.').at(-1)}.${name}`
}

/** A Console line's right-click menu: copy its message (with its further lines), or the selected lines when it was
 * opened on a selection, or hide the lines its source writes, about the object it names or about every object. */
export function LineMenu({
  game,
  at,
  group,
  selection,
  onClose,
}: {
  game: string
  at: { top: number; left: number } | null
  group: readonly Entry[]
  selection: readonly Entry[]
  onClose: () => void
}) {
  const { t } = useLingui()
  const source = sourceOf(group)
  const hide = (about: string, text = source?.text ?? '') => {
    if (source) {
      HideLogSource(game, { ...source, text, object: about }).catch(
        reportError(t`Could not hide that message`),
      )
    }
    onClose()
  }
  const object = source?.object ?? ''
  const name = object.split('/').at(-1) ?? ''
  // A tagged line names its method; an engine message has none to widen to.
  const method = source?.source.startsWith('unity:') ? '' : (source?.source ?? '')
  return (
    <Menu
      open={at !== null}
      onClose={onClose}
      anchorReference="anchorPosition"
      anchorPosition={at ?? undefined}
    >
      <MenuAction
        icon={<Copy size={16} aria-hidden={true} />}
        label={selection.length > 0 ? t`Copy the selected lines` : t`Copy this message`}
        onClick={() => {
          const lines = (selection.length > 0 ? selection : group).map(formatFull).join('\n')
          copyText(lines, selection.length > 0 ? t`Selected lines copied` : t`Message copied`)
          onClose()
        }}
      />
      {object ? (
        <MenuAction
          icon={<EyeOff size={16} aria-hidden={true} />}
          label={t`Hide this message from ${name}`}
          title={object}
          onClick={() => hide(object)}
        />
      ) : null}
      {source ? (
        <MenuAction
          icon={<EyeOff size={16} aria-hidden={true} />}
          label={object ? t`Hide this message from every object` : t`Hide this message`}
          onClick={() => hide('')}
        />
      ) : null}
      {method ? (
        <MenuAction
          icon={<EyeOff size={16} aria-hidden={true} />}
          label={t`Hide every message from ${{ method: shortMethod(method) }}`}
          title={method}
          onClick={() => hide('', '')}
        />
      ) : null}
    </Menu>
  )
}
