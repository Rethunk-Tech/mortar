import type { LoaderRef } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'

// The Streaming page exists only where the loader's game has an overlay component: its settings drive that mod, and
// every other bridge game would show a toggle that does nothing.
export function showsStreaming(loader: LoaderRef | undefined | null): boolean {
  return loader?.overlay === true
}
