import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase } from '@mui/material'
import { LayoutGrid, List } from 'lucide-react'
import type { ViewMode } from './viewMode.ts'

const viewButton = (active: boolean) => ({
  width: 34,
  height: 30,
  borderRadius: '6px',
  bgcolor: active ? 'var(--mortar-hairline-16)' : 'transparent',
  color: active ? 'var(--mortar-ink)' : 'text.secondary',
  '&:hover': { bgcolor: active ? 'var(--mortar-hairline-16)' : 'var(--mortar-hairline-muted)' },
})

/** The grid/list switch at the start of a toolbar. */
function ViewToggle({ value, onChange }: { value: ViewMode; onChange: (next: ViewMode) => void }) {
  const { t } = useLingui()
  return (
    <Box
      role="group"
      aria-label={t`View`}
      sx={{
        display: 'flex',
        p: '3px',
        gap: '2px',
        bgcolor: 'var(--mortar-overlay-30)',
        borderRadius: '8px',
        flexShrink: 0,
      }}
    >
      <ButtonBase
        aria-label={t`Grid view`}
        aria-pressed={value === 'grid'}
        onClick={() => onChange('grid')}
        sx={viewButton(value === 'grid')}
      >
        <LayoutGrid size={15} />
      </ButtonBase>
      <ButtonBase
        aria-label={t`List view`}
        aria-pressed={value === 'list'}
        onClick={() => onChange('list')}
        sx={viewButton(value === 'list')}
      >
        <List size={15} />
      </ButtonBase>
    </Box>
  )
}

export { ViewToggle }
