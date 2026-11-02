import { useLoaded } from '@rethunk/react-load-hooks'
import type { DepSync } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { DependencySync } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'

/** How the project's declared dependencies and its development profile's mods differ; reload after acting on it. */
export function useDependencySync(game: string, dir: string) {
  const loaded = useLoaded<DepSync | null>(() => DependencySync(game, dir), [game, dir], null)
  return { sync: loaded.data, reload: loaded.reload }
}

export const outOfStep = (s: DepSync | null) =>
  s ? (s.missing?.length ?? 0) + (s.undeclared?.length ?? 0) + (s.differs?.length ?? 0) : 0
