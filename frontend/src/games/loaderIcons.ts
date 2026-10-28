import { Blocks, Folder, Package, Puzzle, Shapes, Sprout } from 'lucide-react'
import type { ComponentType } from 'react'

// Puzzle is the launch screen's loader mark; each custom-game loader gets its own, and an unknown one keeps Package.
const ICONS: Record<string, ComponentType<{ size?: number }>> = {
  bepinex5: Puzzle,
  bepinex6: Blocks,
  melonloader: Sprout,
  shimloader: Shapes,
  folder: Folder,
}

/** loaderIcon is the icon for a custom-game loader kind. */
export function loaderIcon(id: string): ComponentType<{ size?: number }> {
  return ICONS[id] ?? Package
}
