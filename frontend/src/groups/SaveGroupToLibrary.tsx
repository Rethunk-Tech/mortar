import { useLingui } from '@lingui/react/macro'
import { useState } from 'react'
import { SaveGroup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savedgroups/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { PromptDialog } from '../shell/PromptDialog.tsx'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { useSavedGroups } from './useSavedGroups.ts'

const MAX_NAME = 60

// Saves a profile group to the game's library under a name that starts as the group's; a name a saved group
// already has asks to replace it.
export function SaveGroupToLibrary({
  open,
  group,
  onClose,
}: {
  open: boolean
  group: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const profileId = useProfiles((s) => s.openId)
  const [busy, run] = usePending()
  const { savedGroups } = useSavedGroups(game, open)
  const [replacing, setReplacing] = useState<string | null>(null)
  const save = (name: string, replace: boolean) =>
    run(
      () =>
        SaveGroup(game, profileId, group, name, replace).then(() => {
          useToasts.getState().push({ kind: 'success', title: t`Saved to library` })
          setReplacing(null)
          onClose()
        }),
      { errorTitle: t`Could not save the group` },
    )
  return (
    <>
      <PromptDialog
        open={open && replacing === null}
        title={t`Save to library`}
        label={t`Saved group name`}
        initial={group}
        maxLength={MAX_NAME}
        confirmLabel={t`Save`}
        busy={busy}
        onCancel={onClose}
        onSubmit={(name) => {
          if (savedGroups.some((x) => x.name.toLowerCase() === name.toLowerCase())) {
            setReplacing(name)
            return
          }
          save(name, false)
        }}
      />
      <ConfirmationDialog
        open={open && replacing !== null}
        title={t`Replace ${{ name: replacing ?? '' }}?`}
        body={t`A saved group with this name already exists.`}
        confirmLabel={t`Replace`}
        busy={busy}
        onCancel={() => setReplacing(null)}
        onConfirm={() => save(replacing ?? '', true)}
      />
    </>
  )
}
