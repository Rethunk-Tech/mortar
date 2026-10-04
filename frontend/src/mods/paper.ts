import { alpha, type Theme } from '@mui/material'

const ACCENT_TINT = 0.14
const ACCENT_BORDER = 0.5
const ACCENT_CHIP = 0.2

const tint = (amount: number) => (th: Theme) => alpha(th.palette.primary.main, amount)

// The accent as a tinted fill, its outline and a stronger chip fill, for update notices.
export const accent = {
  fill: tint(ACCENT_TINT),
  line: tint(ACCENT_BORDER),
  chip: tint(ACCENT_CHIP),
}

export const heading = {
  fontSize: 12,
  fontWeight: 700,
  letterSpacing: '0.06em',
  textTransform: 'uppercase',
  color: 'text.secondary',
} as const
