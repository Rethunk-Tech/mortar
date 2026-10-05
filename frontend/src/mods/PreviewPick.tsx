import { useLingui } from '@lingui/react/macro'
import { Box, Checkbox, FormControlLabel, List, ListItem, ListItemText } from '@mui/material'
import type { ReactNode } from 'react'
import type { GameModPreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { formatPreviewRow } from '../profiles/gameModsFormat.ts'

// A checkbox per mod folder that can be acted on, then the folders that cannot, each with its reason.
export function PreviewPick({
  pickable,
  blocked,
  off,
  onToggle,
  action,
}: {
  pickable: readonly GameModPreview[]
  blocked: readonly GameModPreview[]
  off: ReadonlySet<string>
  onToggle: (folder: string) => void
  action?: (row: GameModPreview) => ReactNode
}) {
  const { t } = useLingui()
  const switchedOff = t`disabled`
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column' }}>
      {pickable.map((m) => (
        <Box key={m.folder} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <FormControlLabel
            sx={{ flex: 1, minWidth: 0, overflowWrap: 'anywhere' }}
            control={
              <Checkbox
                checked={!off.has(m.folder ?? '')}
                onChange={() => onToggle(m.folder ?? '')}
              />
            }
            label={formatPreviewRow(m, switchedOff)}
          />
          {action?.(m)}
        </Box>
      ))}
      <List dense={true} disablePadding={true}>
        {blocked.map((m) => (
          <ListItem key={`${m.name}-${m.folder}`} sx={{ py: 0.5, pl: 1 }}>
            <ListItemText
              primary={formatPreviewRow(m, switchedOff)}
              slotProps={{
                primary: {
                  sx: { fontSize: 14, color: 'text.secondary', overflowWrap: 'anywhere' },
                },
              }}
            />
          </ListItem>
        ))}
      </List>
    </Box>
  )
}
