import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { PackagePlus, Pin, PinOff, Power, PowerOff, Share2, Tag, Trash2, X } from 'lucide-react'
import { useState } from 'react'
import { Create } from '../../bindings/github.com/Rethunk-AI/mortar/internal/bundles/service.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { CopyMods } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { BundleNameDialog } from '../bundles/dialogs.tsx'
import { useProfiles } from '../profiles/store.ts'
import { openShare } from '../share/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useCustomCategories } from './customCategories.ts'
import { modId, visibleUpdates } from './lookup.ts'
import { OtherProfilesDialog } from './OtherProfilesDialog.tsx'
import { CategorySelectionDialog, TagSelectionDialog } from './SelectionBarDialogs.tsx'
import { useSelection } from './selection.ts'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'
import { useLocked } from './useLocked.ts'

const noWrap = { whiteSpace: 'nowrap' } as const

export function SelectionBar({ profileId, mods }: { profileId: string; mods: Mod[] }) {
  const { t } = useLingui()
  const ids = useSelection((s) => s.ids)
  const clear = useSelection((s) => s.clear)
  const setEnabledMany = useMods((s) => s.setEnabledMany)
  const setPinnedMany = useMods((s) => s.setPinnedMany)
  const setSkipVersionMany = useMods((s) => s.setSkipVersionMany)
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
  if (ids.length === 0) {
    return null
  }
  return (
    <>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          px: 2,
          py: 0.75,
          minHeight: 40,
          borderBottom: '1px solid rgba(255,255,255,0.08)',
          flexWrap: 'wrap',
        }}
      >
        <Typography sx={{ fontSize: 13, mr: 0.5, whiteSpace: 'nowrap' }}>{count}</Typography>
        <Button
          size="small"
          variant="outlined"
          disabled={locked}
          startIcon={<Power size={15} />}
          onClick={() => setEnabledMany(selected, true).catch(reportUnexpected)}
          sx={noWrap}
        >
          {t`Enable`}
        </Button>
        <Button
          size="small"
          variant="outlined"
          disabled={locked}
          startIcon={<PowerOff size={15} />}
          onClick={() => setEnabledMany(selected, false).catch(reportUnexpected)}
          sx={noWrap}
        >
          {t`Disable`}
        </Button>
        <Button
          size="small"
          variant="outlined"
          disabled={locked}
          startIcon={<Tag size={15} />}
          onClick={() => setTagOpen(true)}
          sx={noWrap}
        >
          {t`Tag`}
        </Button>
        <Button
          size="small"
          variant="outlined"
          disabled={locked}
          onClick={() => setCategoryOpen(true)}
          sx={noWrap}
        >
          {t`Set category`}
        </Button>
        <Button
          size="small"
          variant="outlined"
          disabled={locked}
          startIcon={
            selected.every(
              (mod) => profile?.entries?.find((entry) => entry.key === mod.key)?.pinned,
            ) ? (
              <PinOff size={15} />
            ) : (
              <Pin size={15} />
            )
          }
          onClick={() => {
            const pinned = !selected.every(
              (mod) => profile?.entries?.find((entry) => entry.key === mod.key)?.pinned,
            )
            setPinnedMany(selected, pinned).catch(reportUnexpected)
          }}
          sx={noWrap}
        >
          {selected.every((mod) => profile?.entries?.find((entry) => entry.key === mod.key)?.pinned)
            ? t`Unpin version`
            : t`Pin version`}
        </Button>
        <Button
          size="small"
          variant="outlined"
          disabled={locked}
          onClick={() =>
            setSkipVersionMany(selected.filter((mod) => latest.has(mod.key))).catch(
              reportUnexpected,
            )
          }
          sx={noWrap}
        >
          {t`Skip current updates`}
        </Button>
        <Button
          size="small"
          variant="outlined"
          disabled={locked}
          onClick={() => setAlsoOpen(true)}
          sx={noWrap}
        >
          {t`Also add to…`}
        </Button>
        <Button
          size="small"
          variant="outlined"
          disabled={locked}
          startIcon={<PackagePlus size={15} />}
          onClick={() => setSaveOpen(true)}
          sx={noWrap}
        >
          {t`Save as bundle…`}
        </Button>
        <Button
          size="small"
          variant="outlined"
          color="error"
          disabled={locked}
          startIcon={<Trash2 size={15} />}
          onClick={() => askRemove(selected)}
          sx={noWrap}
        >
          {t`Remove`}
        </Button>
        <Button
          size="small"
          variant="outlined"
          startIcon={<Share2 size={15} />}
          onClick={() => openShare(profileId, keys)}
          sx={noWrap}
        >
          {t`Share selection`}
        </Button>
        <Button size="small" variant="text" startIcon={<X size={15} />} onClick={clear} sx={noWrap}>
          {t`Clear`}
        </Button>
      </Box>
      <BundleNameDialog
        open={saveOpen}
        title={t`Save selection as bundle`}
        submitLabel={t`Save`}
        errorTitle={t`Could not create the bundle`}
        onClose={() => setSaveOpen(false)}
        onSubmit={async (name) => {
          const game = useProfiles.getState().game?.id ?? ''
          const created = await Create(game, name, profileId, [
            ...new Set(selected.map((mod) => mod.uniqueId)),
          ])
          useToasts.getState().push({ kind: 'success', title: t`Created ${created.name}` })
        }}
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
        uniqueId={selected[0]?.uniqueId ?? ''}
        uniqueIds={[...new Set(selected.map((mod) => mod.uniqueId))]}
        title={t`Also add selected mods to…`}
        confirmLabel={t`Add`}
        onConfirm={async (profiles) => {
          await Promise.all(
            profiles.map((other) =>
              CopyMods(useProfiles.getState().game?.id ?? '', profileId, other.id, [
                ...new Set(selected.map((mod) => mod.uniqueId)),
              ]),
            ),
          )
        }}
      />
    </>
  )
}
