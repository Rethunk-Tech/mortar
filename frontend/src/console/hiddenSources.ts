import { useEffect, useMemo, useState } from 'react'
import type { HiddenLogSource } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { LogSuppressionDefaults } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useSettings } from '../settings/store.ts'

const NONE: readonly HiddenLogSource[] = []
const fetched = new Map<string, Promise<HiddenLogSource[]>>()

// Each game's published defaults, asked for once a run; a failed ask is forgotten, so the next mount asks again.
function defaultsOf(game: string): Promise<HiddenLogSource[]> {
  let p = fetched.get(game)
  if (!p) {
    p = LogSuppressionDefaults(game).then((list) => list ?? [])
    fetched.set(game, p)
  }
  return p
}

export function sameSource(a: HiddenLogSource, b: HiddenLogSource): boolean {
  return (
    a.source === b.source &&
    (a.text ?? '') === (b.text ?? '') &&
    (a.object ?? '') === (b.object ?? '')
  )
}

/** Every source of the game's log lines hidden, each once (the published defaults the player has not shown again,
 * then the player's own), and the defaults, as settings.HiddenLogSources merges them for the bridge. */
export function useHiddenSources(
  game: string,
  read = true,
): {
  hidden: HiddenLogSource[]
  defaults: readonly HiddenLogSource[]
} {
  const [loaded, setLoaded] = useState<{ game: string; items: readonly HiddenLogSource[] }>({
    game: '',
    items: NONE,
  })
  useEffect(() => {
    let live = true
    if (game !== '' && read) {
      defaultsOf(game)
        .then((items) => live && setLoaded({ game, items }))
        .catch(() => fetched.delete(game))
    }
    return () => {
      live = false
    }
  }, [game, read])
  const defaults = loaded.game === game ? loaded.items : NONE
  const own = useSettings((s) => s.games?.[game]?.hiddenLogSources)
  const shown = useSettings((s) => s.games?.[game]?.shownLogSources)
  const hidden = useMemo(
    () => [
      ...defaults.filter(
        (d) =>
          !(
            (shown ?? []).some((s) => sameSource(s, d)) || (own ?? []).some((o) => sameSource(o, d))
          ),
      ),
      ...(own ?? []),
    ],
    [defaults, own, shown],
  )
  return { hidden, defaults }
}
