import type { SxProps, Theme } from '@mui/material'

/** The dark pill that holds a toolbar's toggle buttons. */
export const pillGroupSx: SxProps<Theme> = {
  display: 'flex',
  flexWrap: 'nowrap',
  flexShrink: 0,
  p: '3px',
  gap: '2px',
  bgcolor: 'var(--mortar-overlay-30)',
  borderRadius: '8px',
}

/** One button in that pill. The group's own joined-border styles are cancelled; the selected one takes `selected`. */
export const pillButtonSx = (
  idle: string,
  selected: { bg: string; hover: string },
): SxProps<Theme> => ({
  border: 0,
  borderRadius: '6px !important',
  ml: '0 !important',
  p: 0,
  minWidth: 0,
  fontWeight: 'inherit',
  lineHeight: 'inherit',
  bgcolor: 'transparent',
  color: idle,
  '&:hover': { bgcolor: 'var(--mortar-hairline-muted)' },
  '&.Mui-selected': { bgcolor: selected.bg, color: 'var(--mortar-ink)' },
  '&.Mui-selected:hover': { bgcolor: selected.hover },
  '&.Mui-disabled': { opacity: 0.4, border: 0 },
})
