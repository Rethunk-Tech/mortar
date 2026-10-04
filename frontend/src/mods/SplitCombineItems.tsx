import { useLingui } from '@lingui/react/macro'
import { ListItemText, MenuItem } from '@mui/material'
import type { Entry } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  CombineEntries,
  SplitExtra,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { pushUndoToast } from '../toasts/undo.ts'
import { applyWithUndo, entryFileLabel } from './menu.ts'

export function SplitCombineItems({
  extras,
  siblings,
  locked,
  close,
  gameId,
  profileId,
  entryKey,
  extraLabel,
}: {
  extras: string[]
  siblings: Entry[]
  locked: boolean
  close: () => void
  gameId: string
  profileId: string
  entryKey: string
  extraLabel: (extraKey: string) => string
}) {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const lockedTip = t`Stop the game to change mods.`
  if (extras.length === 0 && siblings.length === 0) {
    return null
  }
  const toast = (title: string) => (undo: () => unknown) => {
    pushUndoToast(push, title, t`Undo`, undo)
  }
  return (
    <>
      {extras.length > 0 ? (
        <MenuItem disabled={true}>
          <ListItemText>{t`Install separately`}</ListItemText>
        </MenuItem>
      ) : null}
      {extras.map((extraKey) => (
        <MenuItem
          key={`split-${extraKey}`}
          disabled={locked}
          sx={{ pl: 4 }}
          onClick={() => {
            close()
            applyWithUndo(
              gameId,
              profileId,
              () => SplitExtra(gameId, profileId, entryKey, extraKey),
              toast(t`Installed separately`),
            ).catch(reportUnexpected)
          }}
        >
          <ListItemText secondary={locked ? lockedTip : undefined}>
            {extraLabel(extraKey)}
          </ListItemText>
        </MenuItem>
      ))}
      {siblings.length > 0 ? (
        <MenuItem disabled={true}>
          <ListItemText>{t`Combine with…`}</ListItemText>
        </MenuItem>
      ) : null}
      {siblings.map((other) => (
        <MenuItem
          key={`combine-${other.key}`}
          disabled={locked}
          sx={{ pl: 4 }}
          onClick={() => {
            close()
            applyWithUndo(
              gameId,
              profileId,
              () => CombineEntries(gameId, profileId, entryKey, other.key),
              toast(t`Combined`),
            ).catch(reportUnexpected)
          }}
        >
          <ListItemText secondary={locked ? lockedTip : undefined}>
            {entryFileLabel(other)}
          </ListItemText>
        </MenuItem>
      ))}
    </>
  )
}
