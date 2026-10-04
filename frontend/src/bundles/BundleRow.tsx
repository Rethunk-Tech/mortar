import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Collapse, Typography } from '@mui/material'
import { ChevronDown, ChevronRight, PackagePlus, Pencil, Trash2, X } from 'lucide-react'
import { useState } from 'react'
import type { Bundle } from '../../bindings/github.com/Rethunk-AI/mortar/internal/bundles/models.ts'
import { RemoveMods } from '../../bindings/github.com/Rethunk-AI/mortar/internal/bundles/service.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { reportError } from '../toasts/report.ts'

const ICON_SIZE = 15

// One saved bundle: its name and holders, a list of its mods (each removable) and the Add, Rename and Delete actions.
export function BundleRow({
  game,
  bundle,
  holderNames,
  onApply,
  onRename,
  onDelete,
  onChanged,
}: {
  game: string
  bundle: Bundle
  holderNames: string
  onApply: () => void
  onRename: () => void
  onDelete: () => void
  onChanged: (bundle: Bundle) => void
}) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  const mods = bundle.mods ?? []
  const modCount = plural(mods.length, { one: '# mod', other: '# mods' })
  return (
    <Box sx={{ p: 1.25, bgcolor: 'var(--mortar-raised)', borderRadius: '6px' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
        <TipIconButton
          label={open ? t`Hide mods of ${bundle.name}` : t`Show mods of ${bundle.name}`}
          aria-expanded={open}
          onClick={() => setOpen(!open)}
        >
          {open ? <ChevronDown size={ICON_SIZE} /> : <ChevronRight size={ICON_SIZE} />}
        </TipIconButton>
        <Typography
          noWrap={true}
          title={bundle.name}
          sx={{ flex: 1, minWidth: 0, fontWeight: 600 }}
        >
          {bundle.name}
        </Typography>
        <TipIconButton label={t`Add ${bundle.name} to a profile`} onClick={onApply}>
          <PackagePlus size={ICON_SIZE} />
        </TipIconButton>
        <TipIconButton label={t`Rename ${bundle.name}`} onClick={onRename}>
          <Pencil size={ICON_SIZE} />
        </TipIconButton>
        <TipIconButton label={t`Delete ${bundle.name}`} color="error" onClick={onDelete}>
          <Trash2 size={ICON_SIZE} />
        </TipIconButton>
      </Box>
      <Typography sx={{ color: 'text.secondary', fontSize: 12 }}>{modCount}</Typography>
      <Typography sx={{ color: 'text.secondary', fontSize: 12 }} noWrap={true} title={holderNames}>
        {t`Profiles: ${holderNames}`}
      </Typography>
      <Collapse in={open} unmountOnExit={true}>
        <Box component="ul" sx={{ m: 0, mt: 0.5, p: 0, listStyle: 'none' }}>
          {mods.map((mod) => (
            <Box
              component="li"
              key={mod.uniqueId}
              sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}
            >
              <Typography
                noWrap={true}
                title={mod.name}
                sx={{ flex: 1, minWidth: 0, fontSize: 13 }}
              >
                {mod.name}
              </Typography>
              <TipIconButton
                label={
                  mods.length === 1
                    ? t`A bundle keeps at least one mod; delete the bundle instead`
                    : t`Remove ${mod.name} from ${bundle.name}`
                }
                disabled={mods.length === 1}
                onClick={() => {
                  RemoveMods(game, bundle.id, [mod.uniqueId])
                    .then(onChanged)
                    .catch(reportError(t`Could not remove the mod from the bundle`))
                }}
              >
                <X size={ICON_SIZE} />
              </TipIconButton>
            </Box>
          ))}
        </Box>
      </Collapse>
    </Box>
  )
}
