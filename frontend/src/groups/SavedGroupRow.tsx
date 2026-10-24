import { useLingui } from '@lingui/react/macro'
import { Box, Collapse, Typography } from '@mui/material'
import { ChevronDown, ChevronRight, PackagePlus, Pencil, Trash2, X } from 'lucide-react'
import { useState } from 'react'
import type { SavedGroup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savedgroups/models.ts'
import { RemoveMods } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savedgroups/service.ts'
import { modsLabel } from '../i18n/counts.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'

const ICON_SIZE = 15

// One saved group: its name and holders, a list of its mods (each removable) and the Add, Rename and Delete actions.
export function SavedGroupRow({
  game,
  group,
  holderNames,
  onApply,
  onRename,
  onDelete,
  onChanged,
}: {
  game: string
  group: SavedGroup
  holderNames: string
  onApply: () => void
  onRename: () => void
  onDelete: () => void
  onChanged: (group: SavedGroup) => void
}) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  const mods = group.mods ?? []
  const modCount = modsLabel(mods.length)
  return (
    <Box sx={{ p: space.gap, bgcolor: 'var(--mortar-raised)', borderRadius: '6px' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
        <TipIconButton
          label={
            open ? t`Hide mods of ${{ name: group.name }}` : t`Show mods of ${{ name: group.name }}`
          }
          aria-expanded={open}
          onClick={() => setOpen(!open)}
        >
          {open ? <ChevronDown size={ICON_SIZE} /> : <ChevronRight size={ICON_SIZE} />}
        </TipIconButton>
        <Typography noWrap={true} title={group.name} sx={{ flex: 1, minWidth: 0, fontWeight: 600 }}>
          {group.name}
        </Typography>
        <TipIconButton label={t`Add ${{ name: group.name }} to a profile`} onClick={onApply}>
          <PackagePlus size={ICON_SIZE} />
        </TipIconButton>
        <TipIconButton label={t`Rename ${{ name: group.name }}`} onClick={onRename}>
          <Pencil size={ICON_SIZE} />
        </TipIconButton>
        <TipIconButton label={t`Delete ${{ name: group.name }}`} color="error" onClick={onDelete}>
          <Trash2 size={ICON_SIZE} />
        </TipIconButton>
      </Box>
      <Typography sx={{ color: 'text.secondary', fontSize: 'small.fontSize' }}>
        {modCount}
      </Typography>
      <Typography
        sx={{ color: 'text.secondary', fontSize: 'small.fontSize' }}
        noWrap={true}
        title={holderNames}
      >
        {t`Profiles: ${holderNames}`}
      </Typography>
      <Collapse in={open} unmountOnExit={true}>
        <Box component="ul" sx={{ m: 0, mt: 0.5, p: 0, listStyle: 'none' }}>
          {mods.map((mod) => (
            <Box
              component="li"
              key={mod.id || mod.entryKey}
              sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}
            >
              <Typography
                noWrap={true}
                title={mod.name}
                sx={{ flex: 1, minWidth: 0, fontSize: 'base.fontSize' }}
              >
                {mod.overlayOf ? t`${{ name: mod.name }} (optional file)` : mod.name}
              </Typography>
              {mod.overlayOf ? null : (
                <TipIconButton
                  label={
                    mods.length === 1
                      ? t`A saved group keeps at least one mod; delete the saved group instead`
                      : t`Remove ${{ name: mod.name }} from ${{ group: group.name }}`
                  }
                  disabled={mods.length === 1}
                  onClick={() => {
                    RemoveMods(game, group.id, [mod.id])
                      .then(onChanged)
                      .catch(reportError(t`Could not remove the mod from the saved group`))
                  }}
                >
                  <X size={ICON_SIZE} />
                </TipIconButton>
              )}
            </Box>
          ))}
        </Box>
      </Collapse>
    </Box>
  )
}
