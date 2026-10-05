import type { Package } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/models.ts'

/** Whether a game's catalog entry lists Thunderstore among its mod sources. */
export function hasThunderstore(game: { sources: string[] | null } | null | undefined): boolean {
  return game?.sources?.includes('thunderstore') === true
}

/** One package of a pack as a list line: its name and version, with `off` when the pack disabled it. */
export function packageLine(p: Package, off: string): string {
  const line = p.version ? `${p.native} ${p.version}` : p.native
  return p.disabled ? `${line} · ${off}` : line
}
