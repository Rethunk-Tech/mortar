import { useLingui } from '@lingui/react/macro'
import { useCallback, useEffect, useRef, useState } from 'react'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { isTypingTarget } from '../settings/shortcuts.ts'
import { useSettings } from '../settings/store.ts'
import { TYPE_RESET_MS, typedKey, typedMatch } from '../shell/typeahead.ts'
import { readStored, writeStored } from '../shell/useStoredState.ts'
import { listedGames } from './hidden.ts'
import { gameOrder } from './order.ts'

type Game = GameInfo

// The list is virtual: with every catalog game mounted, scrolling through them all kept each row's decoded art.
export type Item =
  | { kind: 'heading'; key: string; id: string; label: string; count: number; collapsed: boolean }
  | { kind: 'game'; key: string; game: Game }

export function listItems(
  groups: { id: string; label: string; games: Game[] }[],
  collapsed: ReadonlySet<string>,
): Item[] {
  const out: Item[] = []
  for (const { id, label, games } of groups) {
    if (games.length > 0) {
      const folded = collapsed.has(id)
      out.push({
        kind: 'heading',
        key: `h:${id}`,
        id,
        label,
        count: games.length,
        collapsed: folded,
      })
      if (!folded) {
        for (const game of games) {
          out.push({ kind: 'game', key: game.id, game })
        }
      }
    }
  }
  return out
}

const COLLAPSED_KEY = 'mortar.gameSelectCollapsed'

const isStrings = (v: unknown): v is string[] =>
  Array.isArray(v) && v.every((x) => typeof x === 'string')

// The sections the viewer folded away, kept across visits.
export function useCollapsedSections() {
  const [collapsed, setCollapsed] = useState<ReadonlySet<string>>(
    () => new Set(readStored<string[]>(COLLAPSED_KEY, [], isStrings)),
  )
  const save = useCallback((next: ReadonlySet<string>) => {
    writeStored(COLLAPSED_KEY, [...next])
    setCollapsed(next)
  }, [])
  const toggle = useCallback(
    (id: string) =>
      setCollapsed((prev) => {
        const next = new Set(prev)
        if (!next.delete(id)) {
          next.add(id)
        }
        writeStored(COLLAPSED_KEY, [...next])
        return next
      }),
    [],
  )
  return { collapsed, toggle, expandAll: useCallback(() => save(new Set()), [save]) }
}

// Typing the name of a hidden game, or of one in a folded section, first shows the hidden games or unfolds the
// sections, then scrolls to the game once the list holds it.
export function useFindGame(args: {
  hidden: readonly string[] | null | undefined
  showHidden: boolean
  setShowHidden: (show: boolean) => void
  listed: (id: string) => boolean
  reveal: (id: string) => void
  unfold: () => boolean
}) {
  const latest = useRef(args)
  latest.current = args
  const [want, setWant] = useState('')
  useEffect(() => {
    if (want !== '' && latest.current.listed(want)) {
      latest.current.reveal(want)
      setWant('')
    }
  }, [want])
  return (id: string) => {
    const away = Boolean(args.hidden?.includes(id)) && !args.showHidden
    if (away) {
      args.setShowHidden(true)
    }
    if (away || (!args.listed(id) && args.unfold())) {
      setWant(id)
      return
    }
    args.reveal(id)
  }
}

// Typing a game's name anywhere on the page jumps to it. The listener reads the list through a ref so the letters
// typed so far survive the page's re-renders.
export function useTypeToScroll(games: Game[], reveal: (id: string) => void) {
  const list = useRef({ games, reveal })
  list.current = { games, reveal }
  useEffect(() => {
    let typed = ''
    let at = 0
    const onKey = (e: KeyboardEvent) => {
      const { target } = e
      if (
        e.ctrlKey ||
        e.metaKey ||
        e.altKey ||
        e.key.length !== 1 ||
        isTypingTarget(target instanceof HTMLElement ? target : null) ||
        document.querySelector('[role="dialog"], [role="menu"]')
      ) {
        return
      }
      const fresh = e.timeStamp - at > TYPE_RESET_MS
      if (!typedKey(e.key)) {
        // A space inside a name being typed would otherwise press the row that typing just focused.
        if (e.key === ' ' && !fresh) {
          e.preventDefault()
        }
        return
      }
      typed = fresh ? e.key : typed + e.key
      at = e.timeStamp
      const hit = typedMatch(list.current.games, typed, (g) => g.name)
      if (hit) {
        list.current.reveal(hit.id)
      }
    }
    globalThis.addEventListener('keydown', onKey)
    return () => globalThis.removeEventListener('keydown', onKey)
  }, [])
}

// Recently played, then games found on this computer, then the rest of the catalog, most of which is not set up here.
export function useSections(
  games: Game[],
  hiddenIds: readonly string[] | null | undefined,
  showHidden: boolean,
  collapsed: ReadonlySet<string>,
) {
  const { t } = useLingui()
  const lastGame = useSettings((s) => s.lastGame)
  const played = useSettings((s) => s.lastPlayed)
  const ordered = gameOrder(listedGames(games, hiddenIds, showHidden), lastGame, played)
  const { recent } = ordered
  const found = ordered.rest.filter((g) => g.available && g.installed)
  const rest = ordered.rest.filter((g) => !(g.available && g.installed))
  const items = listItems(
    [
      { id: 'recent', label: t`Recently played`, games: recent },
      { id: 'found', label: t`On this computer`, games: found },
      {
        id: 'rest',
        label: recent.length + found.length > 0 ? t`Other games` : t`All games`,
        games: rest,
      },
    ],
    collapsed,
  )
  return { rest, items }
}
