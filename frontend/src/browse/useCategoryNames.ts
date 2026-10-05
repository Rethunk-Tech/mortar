import { useEffect, useState } from 'react'

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
  const [names, setNames] = useState<string[]>([])
  useEffect(() => {
    if (skip || source === '') {
      setNames([])
      return
    }
    let cancelled = false
    categories(game, source)
      .then((next) => !cancelled && setNames(next))
      .catch(() => !cancelled && setNames([]))
    return () => {
      cancelled = true
    }
  }, [categories, game, source, skip])
  return names
}

export { useCategoryNames }
