import { useLingui } from '@lingui/react/macro'
import { Merge, Split } from 'lucide-react'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  CombineEntries,
  SplitExtra,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { MenuHeading } from '../shell/TitleMenu.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { pushUndoToast } from '../toasts/undo.ts'
import { applyWithUndo, entryFileLabel, ICON_SIZE } from './menu.ts'

export function SplitCombineItems({
  extras,
  siblings,
  locked,
  close,
  gameId,
  profileId,
  entryKey,
  extraLabel,
  extraArchive,
}: {
  extras: string[]
  siblings: Entry[]
  locked: boolean
  close: () => void
  gameId: string
  profileId: string
  entryKey: string
  extraLabel: (extraKey: string) => string
  extraArchive: (extraKey: string) => string
}) {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const lockedTip = t`Stop the game to change mods.`
  if (extras.length === 0 && siblings.length === 0) {
    return null
  }
  const toast = (title: string) => (undo: () => unknown) => {
    pushUndoToast(push, title, { label: t`Undo`, run: undo, profileId })
  }
  return (
    <>
      {extras.length > 0 ? <MenuHeading>{t`Install separately`}</MenuHeading> : null}
      {extras.map((extraKey) => (
        <MenuAction
          key={`split-${extraKey}`}
          icon={<Split size={ICON_SIZE} />}
          label={extraLabel(extraKey)}
          detail={extraArchive(extraKey) || undefined}
          disabled={locked}
          reason={lockedTip}
          onClick={() => {
            close()
            applyWithUndo(
              gameId,
              profileId,
              () => SplitExtra(gameId, profileId, entryKey, extraKey),
              toast(t`Installed separately`),
            ).catch(reportUnexpected)
          }}
        />
      ))}
      {siblings.length > 0 ? <MenuHeading>{t`Combine with`}</MenuHeading> : null}
      {siblings.map((other) => (
        <MenuAction
          key={`combine-${other.key}`}
          icon={<Merge size={ICON_SIZE} />}
          label={entryFileLabel(other)}
          disabled={locked}
          reason={lockedTip}
          onClick={() => {
            close()
            applyWithUndo(
              gameId,
              profileId,
              () => CombineEntries(gameId, profileId, entryKey, other.key),
              toast(t`Combined`),
            ).catch(reportUnexpected)
          }}
        />
      ))}
    </>
  )
}
