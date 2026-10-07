import { useLoaded } from '../shell/useLoaded.ts'

// The source's category names; empty while loading, when the source has none, or when the lookup fails.
function useCategoryNames({
  categories,
  game,
  source,
  skip,
}: {
  categories: (game: string, source: string) => Promise<string[]>
  game: string
  source: string
  skip: boolean
}): string[] {
  const { data: names } = useLoaded<string[]>(
    skip || source === '' ? null : () => categories(game, source),
    [categories, game, source, skip],
    [],
  )
  return names
}

export { useCategoryNames }
