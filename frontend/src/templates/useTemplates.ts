import { useLingui } from '@lingui/react/macro'
import { sameValue } from '@rethunk/react-load-hooks'
import { useCallback, useEffect } from 'react'
import { create } from 'zustand'
import type { Template } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/templates/models.ts'
import { Templates } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/templates/service.ts'
import { reportError } from '../toasts/report.ts'

const NONE: Template[] = []

// Each game's templates, shared: the readers that turn on together (a menu's several template items) make one read.
const useTemplateStore = create<{ byGame: Record<string, Template[]> }>(() => ({ byGame: {} }))
const reading = new Map<string, Promise<void>>()
const readAt = new Map<string, number>()
// A read this recent stands for the next reader: the switcher menu reads as it opens, New profile a moment later.
const FRESH_MS = 2000

function read(game: string, force: boolean): Promise<void> {
  const inFlight = reading.get(game)
  if (inFlight && !force) {
    return inFlight
  }
  if (!force && Date.now() - (readAt.get(game) ?? 0) < FRESH_MS) {
    return Promise.resolve()
  }
  const next = Templates(game)
    .then((found) =>
      useTemplateStore.setState((s) => {
        const prev = s.byGame[game]
        const list = found?.length ? found : NONE
        // An unchanged answer keeps the list, so a menu that reads the templates does not redraw it.
        return prev && sameValue(prev, list) ? s : { byGame: { ...s.byGame, [game]: list } }
      }),
    )
    .finally(() => {
      reading.delete(game)
      readAt.set(game, Date.now())
    })
  reading.set(game, next)
  return next
}

/** Reads the game's templates for every reader, unless a read is under way or just ended. */
export const readTemplates = (game: string) => read(game, false)

/** The game's templates, read when `active` turns on and again on `reload`. */
export function useTemplates(game: string, active: boolean) {
  const { t } = useLingui()
  const templates = useTemplateStore((s) => s.byGame[game] ?? NONE)
  const reload = useCallback(
    () =>
      game
        ? read(game, true).catch(reportError(t`Could not read the templates`))
        : Promise.resolve(),
    [game, t],
  )
  useEffect(() => {
    if (active && game) {
      read(game, false).catch(reportError(t`Could not read the templates`))
    }
  }, [active, game, t])
  return { templates, reload }
}
