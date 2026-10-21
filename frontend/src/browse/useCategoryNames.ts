import { useMemo } from 'react'
import { useLoaded } from '../shell/useLoaded.ts'
import { errorMessage } from '../toasts/report.ts'

// The source's category names, empty while loading, when the source has none, or when the lookup fails; `why` is the
// failure's reason, so the pickers can say it instead of "no categories".
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
}): { names: string[]; why: string } {
  const { data: names, error } = useLoaded<string[]>(
    skip || source === '' ? null : () => categories(game, source),
    [categories, game, source, skip],
    [],
  )
  // One object while its parts are, so the memoised filter row skips a render when results arrive.
  return useMemo(() => ({ names, why: error ? errorMessage(error) : '' }), [names, error])
}

export { useCategoryNames }
