import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { useEffect, useState } from 'react'
import type { Update } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type { Profile } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { UpdateEntry } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useProfiles } from '../../profiles/store.ts'
import { useQueue } from '../../queue/store.ts'
import { useToasts } from '../../toasts/store.ts'
import { sameId } from '../lookup.ts'
import { OtherProfilesDialog } from '../OtherProfilesDialog.tsx'

export function PropagateUpdate({
  profile,
  update,
  onDone,
}: {
  profile: Profile
  update: Update
  onDone: () => void
}) {
  const { t } = useLingui()
  const items = useQueue((s) => s.state.items)
  const [newKey, setNewKey] = useState('')
  useEffect(() => {
    const done = items.find(
      (item) =>
        item.state === 'done' &&
        item.kind === 'update' &&
        item.profileId === profile.id &&
        item.name === update.name &&
        item.version === update.version,
    )
    if (!done) {
      return
    }
    const next = useProfiles
      .getState()
      .profiles.find((candidate) => candidate.id === profile.id)
      ?.entries?.find((entry) =>
        entry.mods?.some((mod) => sameId(mod.uniqueId, update.uniqueId)),
      )?.key
    if (next) {
      setNewKey(next)
    }
  }, [items, profile.id, update])
  return (
    <OtherProfilesDialog
      open={newKey !== ''}
      onClose={onDone}
      game={useProfiles.getState().game?.id ?? ''}
      currentProfileId={profile.id}
      uniqueId={update.uniqueId}
      title={t`Update ${update.name} in other profiles`}
      confirmLabel={t`Update profiles`}
      update={{ oldKey: update.key }}
      onConfirm={async (profiles, pinned) => {
        const game = useProfiles.getState().game?.id ?? ''
        await Promise.all(profiles.map((other) => UpdateEntry(game, other.id, update.key, newKey)))
        useToasts.getState().push({
          kind: 'success',
          title: t`${plural(profiles.length, { one: 'Updated in # profile', other: 'Updated in # profiles' })}`,
          ...(pinned.length > 0
            ? { body: pinned.map((other) => t`pinned in ${other.name}`).join(', ') }
            : {}),
        })
      }}
    />
  )
}
