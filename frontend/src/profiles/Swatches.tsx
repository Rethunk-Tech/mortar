import { ToggleButton, ToggleButtonGroup } from '@mui/material'
import { Check } from 'lucide-react'
import type { ReactNode } from 'react'
import { Tooltip } from '../shell/Tooltip.tsx'
import { colorHex, PROFILE_COLORS } from './appearance.ts'
import { useColorName } from './useColorName.ts'

const COLOR_SWATCH_PX = 32
const ICON_SWATCH_PX = 36

// The group's joined-button borders and radii are dropped: swatches stand apart, each with its own outline.
const groupSx = (px: number, round: boolean) => ({
  display: 'flex',
  flexWrap: 'wrap',
  gap: 0.75,
  '& .MuiToggleButtonGroup-grouped.MuiToggleButton-root': {
    m: 0,
    p: 0,
    border: 0,
    width: px,
    height: px,
    borderRadius: round ? '50%' : '6px',
    // The theme's focus ring wins over the selection outline.
    '&:not(.Mui-focusVisible)': { outline: '2px solid transparent', outlineOffset: 2 },
    '&.Mui-selected:not(.Mui-focusVisible)': { outlineColor: 'var(--mortar-ink)' },
  },
})

// Swatches is a single-choice row of swatches; choosing the selected one again clears the choice to ''.
function Swatches({
  label,
  value,
  options,
  onChange,
  round,
}: {
  label: string
  value: string
  options: readonly { value: string; name: string; face: ReactNode; color?: string | undefined }[]
  onChange: (next: string) => void
  round: boolean
}) {
  const px = round ? COLOR_SWATCH_PX : ICON_SWATCH_PX
  return (
    <ToggleButtonGroup
      exclusive={true}
      aria-label={label}
      value={value === '' ? null : value}
      onChange={(_, next: string | null) => onChange(next ?? '')}
      sx={groupSx(px, round)}
    >
      {options.map((o) => (
        <Tooltip key={o.value} title={o.name}>
          <ToggleButton
            value={o.value}
            aria-label={o.name}
            sx={{
              ...(round
                ? { '&, &:hover, &.Mui-selected, &.Mui-selected:hover': { bgcolor: o.color } }
                : {
                    '&.Mui-selected, &.Mui-selected:hover': {
                      bgcolor: 'var(--mortar-hairline-12)',
                    },
                  }),
            }}
          >
            {o.face}
          </ToggleButton>
        </Tooltip>
      ))}
    </ToggleButtonGroup>
  )
}

export function ColorSwatches({
  label,
  value,
  onChange,
}: {
  label: string
  value: string
  onChange: (next: string) => void
}) {
  const colorName = useColorName()
  const options = PROFILE_COLORS.map((token) => ({
    value: token,
    name: colorName(token),
    color: colorHex(token),
    face: value === token ? <Check size={16} color="#1a1a1a" aria-hidden={true} /> : null,
  }))
  return <Swatches label={label} value={value} options={options} onChange={onChange} round={true} />
}

export function IconSwatches({
  label,
  value,
  options,
  onChange,
}: {
  label: string
  value: string
  options: readonly { value: string; name: string; face: ReactNode }[]
  onChange: (next: string) => void
}) {
  return (
    <Swatches label={label} value={value} options={options} onChange={onChange} round={false} />
  )
}
