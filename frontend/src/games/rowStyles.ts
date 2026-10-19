export const ROW_PX = 168
// How much taller the row under the pointer, or holding focus, stands.
const ROW_GROW_PX = 72
const HOVER_MS = 450
const HOVER_EASE = 'ease-in-out'

// The row under the pointer, or holding focus, grows taller and shows more of its art.
export const hoverFocus = {
  // Keyboard focus only: a clicked Play button keeps focus, which would leave its row grown under a pointer elsewhere.
  '& [data-tile]:has(:focus-visible)': { minHeight: ROW_PX + ROW_GROW_PX },
  '@media (hover: hover)': {
    '& [data-tile]:hover': { minHeight: ROW_PX + ROW_GROW_PX },
  },
  '@media (prefers-reduced-motion: no-preference)': {
    '& [data-tile]': { transition: `min-height ${HOVER_MS}ms ${HOVER_EASE}` },
  },
} as const

// Rows have a fixed height, so the list never measures them. A row that grows under the pointer overlays the next
// one instead of resizing the list.
export const listItemSx = {
  position: 'absolute',
  top: 0,
  left: 0,
  width: '100%',
  '&:hover, &:focus-within': { zIndex: 1 },
} as const
