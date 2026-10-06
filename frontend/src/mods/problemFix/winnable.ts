import { sameId } from '../lookup.ts'

interface PackDeps {
  id: string
  needs?: readonly string[] | null
  contentPackFor?: string | null
}

// winnable is the indexes of packIds that can be made to load after all the others. SMAPI already loads a pack after
// any installed mod it lists, optional or not, and after the framework it is a content pack for, so a pack that
// another of them waits for can never load last.
export function winnable(mods: readonly PackDeps[], packIds: readonly string[]): number[] {
  const waitsFor = (id: string) => {
    const m = mods.find((x) => sameId(x.id, id))
    return [...(m?.needs ?? []), ...(m?.contentPackFor ? [m.contentPackFor] : [])]
  }
  return packIds.flatMap((winner, index) =>
    packIds.some(
      (other) => !sameId(other, winner) && waitsFor(other).some((d) => sameId(d, winner)),
    )
      ? []
      : [index],
  )
}
