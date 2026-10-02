import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { PackagePlus, Power, PowerOff, Share2, Trash2, X } from 'lucide-react'
import { useState } from 'react'
import { Create } from '../../bindings/github.com/Rethunk-AI/mortar/internal/bundles/service.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { CopyMods } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { BundleNameDialog } from '../bundles/dialogs.tsx'
import { useProfiles } from '../profiles/store.ts'
import { openShare } from '../share/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { modId } from './lookup.ts'
import { OtherProfilesDialog } from './OtherProfilesDialog.tsx'
import { useSelection } from './selection.ts'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

const noWrap = { whiteSpace: 'nowrap' } as const

export function SelectionBar({ profileId, mods }: { profileId: string; mods: Mod[] }) {
  const { t } = useLingui()
  const ids = useSelection((s) => s.ids)
  const clear = useSelection((s) => s.clear)
  const setEnabledMany = useMods((s) => s.setEnabledMany)
  const askRemove = useMods((s) => s.askRemove)
  const locked = useLocked()
  const [alsoOpen, setAlsoOpen] = useState(false)
  const [saveOpen, setSaveOpen] = useState(false)
  if (ids.length < 2) {
    return null
  }
  const selected = mods.filter((m) => ids.includes(modId(m)))
  const keys = [...new Set(selected.map((m) => m.key))]
  const count = plural(ids.length, { one: '# mod selected', other: '# mods selected' })
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
