import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { useState } from 'react'
import { Create } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/bundles/service.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { CopyMods } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { BundleNameDialog } from '../bundles/dialogs.tsx'
import { useProfiles } from '../profiles/store.ts'
import { openShare } from '../share/store.ts'
import { useToasts } from '../toasts/store.ts'
import { useCustomCategories } from './customCategories.ts'
import { modId, visibleUpdates } from './lookup.ts'
import { OtherProfilesDialog } from './OtherProfilesDialog.tsx'
import { SelectionBarActions } from './SelectionBarActions.tsx'
import { CategorySelectionDialog, TagSelectionDialog } from './SelectionBarDialogs.tsx'
import { useSelection } from './selection.ts'
import { skipSourceGroups } from './skipSources.ts'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'
import { useLocked } from './useLocked.ts'

function SaveSelectionDialog({
  open,
  onClose,
  profileId,
  selected,
}: {
  open: boolean
  onClose: () => void
  profileId: string
  selected: Mod[]
}) {
  const { t } = useLingui()
  return (
    <BundleNameDialog
      open={open}
      title={t`Save selection as bundle`}
      submitLabel={t`Save`}
      errorTitle={t`Could not create the bundle`}
      onClose={onClose}
      onSubmit={async (name) => {
        const game = useProfiles.getState().game?.id ?? ''
        const created = await Create(game, name, profileId, [
          ...new Set(selected.map((mod) => mod.id)),
        ])
        useToasts.getState().push({ kind: 'success', title: t`Created ${{ name: created.name }}` })
      }}
    />
  )
}

function SelectionBar({ profileId, mods }: { profileId: string; mods: Mod[] }) {
  const { t } = useLingui()
  const ids = useSelection((s) => s.ids)
  const clear = useSelection((s) => s.clear)
  const setEnabledMany = useMods((s) => s.setEnabledMany)
  const setPinnedMany = useMods((s) => s.setPinnedMany)
  const setSkipVersionMany = useMods((s) => s.setSkipVersionMany)
  const setSkipSourceMany = useMods((s) => s.setSkipSourceMany)
  const setCategoryMany = useMods((s) => s.setCategoryMany)
  const setTagMany = useMods((s) => s.setTagMany)
  const askRemove = useMods((s) => s.askRemove)
  const locked = useLocked()
  const [alsoOpen, setAlsoOpen] = useState(false)
  const [saveOpen, setSaveOpen] = useState(false)
  const [tagOpen, setTagOpen] = useState(false)
  const [categoryOpen, setCategoryOpen] = useState(false)
  const [tag, setTag] = useState('')
  const [addTag, setAddTag] = useState(true)
  const [category, setCategory] = useState('')
  const selected = mods.filter((m) => ids.includes(modId(m)))
  const keys = [...new Set(selected.map((m) => m.key))]
  const count = plural(ids.length, { one: '# mod selected', other: '# mods selected' })
  const updates = visibleUpdates(
    useUpdates((s) => s.updates),
    useProfiles.getState().profiles.find((p) => p.id === profileId),
  )
  const latest = new Map(updates.map((update) => [update.key, update.version]))
  const categories = useCustomCategories((s) => s.categories)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === profileId))
  const tags = [...new Set((profile?.entries ?? []).flatMap((entry) => entry.tags ?? []))].sort()
  // One selected mod is covered by the details panel and its menu; the bar is for acting on several.
  const multi = ids.length > 1
  return (
    <>
      {multi ? (
        <SelectionBarActions
          selected={selected}
          profile={profile}
          latest={latest}
          locked={locked}
          count={count}
          setEnabledMany={setEnabledMany}
          setPinnedMany={setPinnedMany}
          setSkipVersionMany={setSkipVersionMany}
          sourceGroups={skipSourceGroups(selected, updates, profile?.entries)}
          setSkipSourceMany={setSkipSourceMany}
          askRemove={askRemove}
          openAlso={() => setAlsoOpen(true)}
          openSave={() => setSaveOpen(true)}
          openTag={() => setTagOpen(true)}
          openCategory={() => setCategoryOpen(true)}
          share={() => openShare(profileId, keys)}
          clear={clear}
        />
      ) : null}
      <SaveSelectionDialog
        open={saveOpen}
        onClose={() => setSaveOpen(false)}
        profileId={profileId}
        selected={selected}
      />
      <TagSelectionDialog
        open={tagOpen}
        onClose={() => setTagOpen(false)}
        mods={selected}
        tags={tags}
        setTag={setTag}
        tag={tag}
        addTag={addTag}
        setAddTag={setAddTag}
        setTagMany={setTagMany}
      />
      <CategorySelectionDialog
        open={categoryOpen}
        onClose={() => setCategoryOpen(false)}
        mods={selected}
        categories={categories}
        category={category}
        setCategory={setCategory}
        setCategoryMany={setCategoryMany}
      />
      <OtherProfilesDialog
        open={alsoOpen}
        onClose={() => setAlsoOpen(false)}
        game={useProfiles.getState().game?.id ?? ''}
        currentProfileId={profileId}
        id={selected[0]?.id ?? ''}
        ids={[...new Set(selected.map((mod) => mod.id))]}
        title={t`Also add selected mods to…`}
        confirmLabel={t`Add`}
        onConfirm={async (profiles) => {
          await Promise.all(
            profiles.map((other) =>
              CopyMods(useProfiles.getState().game?.id ?? '', profileId, other.id, [
                ...new Set(selected.map((mod) => mod.id)),
              ]),
            ),
          )
        }}
      />
    </>
  )
}

export { SelectionBar }
