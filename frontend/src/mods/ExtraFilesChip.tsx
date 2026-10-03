import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Chip, Tooltip } from '@mui/material'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { entryOf, nexusIdOf } from './lookup.ts'
import { extraFileLabel, useContextMenu } from './menu.ts'
import { useNexusEntry } from './nexusDetails.ts'

export function ExtraFilesChip({ mod, profile }: { mod: Mod; profile: Profile }) {
  const { i18n } = useLingui()
  const entry = entryOf(profile, mod.key)
  const extras = entry?.extraStoreKeys ?? []
  const files = useNexusEntry(nexusIdOf(profile, mod))?.details?.files
  const openMenu = useContextMenu((s) => s.open)
  if (extras.length === 0) {
    return null
  }
  const labels = extras.map((key) => extraFileLabel(entry, key, files))
  const title = labels.join('\n')
  const label = i18n._(plural(extras.length, { one: '+# file', other: '+# files' }))
  return (
    <Tooltip title={title}>
      <Chip
        size="small"
        label={label}
        onClick={(e) => {
          e.stopPropagation()
          openMenu(mod, { el: e.currentTarget })
        }}
        sx={{
          height: 20,
          fontSize: 11,
          bgcolor: 'action.selected',
          color: 'text.secondary',
          '& .MuiChip-label': { px: 0.75 },
        }}
      />
    </Tooltip>
  )
}
