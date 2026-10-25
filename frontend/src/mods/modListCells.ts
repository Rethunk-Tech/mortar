const ellipsis = { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } as const

// A row's cells are plain elements styled from the row: a list redraws many rows at once (a freeze, a reload), and a
// TableCell or Box is three components more per cell each time.
export const cellsSx = {
  '& > td': {
    p: 0,
    border: 0,
    typography: 'body2',
    fontSize: 'inherit',
    color: 'inherit',
    textAlign: 'left',
  },
  '& .cell-line': { display: 'flex', alignItems: 'center', gap: 0.5, minWidth: 0 },
  '& .cell-tile': { gap: 1 },
  '& .cell-text': { ...ellipsis, minWidth: 0 },
  '& .cell-value': { ...ellipsis, color: 'text.secondary', fontVariantNumeric: 'tabular-nums' },
  '& .cell-value.cell-accent': { color: 'primary.main' },
  '& .cell-name': { ...ellipsis, fontWeight: 500 },
  '& .cell-depth': { display: 'flex', flexShrink: 0, color: 'text.secondary' },
  '& .cell-version': { ...ellipsis, color: 'text.secondary' },
  '& .cell-version .cell-text': { fontVariantNumeric: 'tabular-nums' },
  '& .cell-status': { fontSize: 'base.fontSize', whiteSpace: 'nowrap', color: 'text.primary' },
  '& .cell-status.cell-dim': { color: 'text.secondary' },
  '& .cell-status .cell-line': { gap: 0.75 },
  '& .cell-overrides': {
    ...ellipsis,
    flexShrink: 0,
    fontSize: 'small.fontSize',
    fontWeight: 400,
    color: 'text.secondary',
  },
} as const
