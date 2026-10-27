// Every dialog's paper: inset from the window, rounded, a hairline border, its content laid out as a column.
export const paperSx = {
  maxWidth: 'calc(100% - 32px)',
  maxHeight: 'calc(100% - 64px)',
  m: 2,
  borderRadius: '10px',
  border: '1px solid var(--mortar-hairline)',
  display: 'flex',
  flexDirection: 'column',
  overflow: 'hidden',
} as const
