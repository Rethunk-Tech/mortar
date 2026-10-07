// The spacing scale, set once per Density level as CSS variables on :root. Every tab page, panel, toolbar, table and
// menu reads these through `space` rather than hand-setting pixels, so a level change moves them all together.
const levels = {
  comfortable: { gutter: 16, gap: 8, pad: 16, row: 36, menuY: 6, control: 36 },
  compact: { gutter: 12, gap: 6, pad: 12, row: 30, menuY: 4, control: 32 },
} as const

export type DensityLevel = keyof typeof levels

export const densityVars = (level: DensityLevel) => {
  const v = levels[level]
  return {
    '--m-gutter': `${v.gutter}px`,
    '--m-gap': `${v.gap}px`,
    '--m-pad': `${v.pad}px`,
    '--m-row': `${v.row}px`,
    '--m-menu-y': `${v.menuY}px`,
    '--m-control': `${v.control}px`,
  }
}

export const space = {
  // A page's left and right edge, and the inset of a controls row.
  gutter: 'var(--m-gutter)',
  // Between controls in a row, and between a row and what follows.
  gap: 'var(--m-gap)',
  // Inside a card or panel.
  pad: 'var(--m-pad)',
  // A table or list row.
  row: 'var(--m-row)',
  // Above and below a menu item's text.
  menuY: 'var(--m-menu-y)',
  // Buttons, dropdowns and fields that share a row.
  control: 'var(--m-control)',
} as const
