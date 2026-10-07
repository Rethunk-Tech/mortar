// The first non-empty line of a profile's notes, for the one-line strip in the tab header.
export function notesFirstLine(notes: string): string {
  return (
    notes
      .split('\n')
      .map((line) => line.trim())
      .find((line) => line !== '') ?? ''
  )
}

// One height for every control a tab puts in the header's page-actions slot (icon buttons, buttons and both halves of
// a split button). Set from the slot, so a tab's own sizes cannot drift from its neighbours.
export const PAGE_ACTION_PX = 34

export const pageActionsSx = {
  '& .MuiButton-root, & .MuiIconButton-root': {
    height: PAGE_ACTION_PX,
    minHeight: PAGE_ACTION_PX,
    boxSizing: 'border-box',
  },
  '& .MuiIconButton-root': { width: PAGE_ACTION_PX },
} as const
