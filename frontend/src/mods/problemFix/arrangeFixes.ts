/** How much a fix weighs: the suggested fix, another real fix such as Disable, or a quiet one such as Dismiss. */
export type FixWeight = 'main' | 'other' | 'quiet'

const RANK: Record<FixWeight, number> = { main: 0, other: 1, quiet: 2 }

/** A problem row's fixes by weight, keeping each weight's own order: the heaviest is the button, the other real
 * fixes and then the quiet ones fill its menu. With only quiet fixes there is no button and they show as text. */
export function arrangeFixes<T extends { weight: FixWeight }>(
  fixes: readonly T[],
): { main: T | undefined; menu: T[]; quiet: T[] } {
  const sorted = [...fixes].sort((a, b) => RANK[a.weight] - RANK[b.weight])
  const loud = sorted.filter((fix) => fix.weight !== 'quiet')
  return {
    main: loud[0],
    menu: loud.slice(1),
    quiet: sorted.filter((fix) => fix.weight === 'quiet'),
  }
}
