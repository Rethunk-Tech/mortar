import { CARD_MIN_PX, GAP_PX } from './browseConstants.ts'

interface SkeletonSize {
  width: number
  height: number
  grid: boolean
  rowPx: number
}

// Cells that fill a box the way the results grid (auto-fill columns) or list lays them out; one row until measured.
function skeletonCount({ width, height, grid, rowPx }: SkeletonSize): number {
  const columns = grid ? Math.max(1, Math.floor((width + GAP_PX) / (CARD_MIN_PX + GAP_PX))) : 1
  const rows = Math.max(1, Math.ceil((height + GAP_PX) / (rowPx + GAP_PX)))
  return columns * rows
}

export { skeletonCount }
