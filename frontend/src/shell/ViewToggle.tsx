import { useLingui } from '@lingui/react/macro'
import { ToggleButton, ToggleButtonGroup } from '@mui/material'
import { LayoutGrid, List } from 'lucide-react'
import { space } from '../theme/density.ts'
import { pillButtonSx, pillGroupSx } from './pillToggle.ts'
import type { ViewMode } from './viewMode.ts'

const HAIRLINE = 'var(--mortar-hairline-16)'
const viewButton = {
  ...pillButtonSx('text.secondary', { bg: HAIRLINE, hover: HAIRLINE }),
  width: 34,
  height: `calc(${space.control} - 6px)`,
}

/** The grid/list switch at the start of a toolbar. */
function ViewToggle({ value, onChange }: { value: ViewMode; onChange: (next: ViewMode) => void }) {
  const { t } = useLingui()
  return (
    <ToggleButtonGroup
      exclusive={true}
      value={value}
      aria-label={t`View`}
      onChange={(_, next: ViewMode | null) => {
        if (next !== null) {
          onChange(next)
        }
      }}
      sx={pillGroupSx}
    >
      <ToggleButton value="grid" aria-label={t`Grid view`} sx={viewButton}>
        <LayoutGrid size={15} />
      </ToggleButton>
      <ToggleButton value="list" aria-label={t`List view`} sx={viewButton}>
        <List size={15} />
      </ToggleButton>
    </ToggleButtonGroup>
  )
}

export { ViewToggle }
