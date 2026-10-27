import { type Dispatch, type SetStateAction, useEffect, useState } from 'react'
import type {
  HistoryDiff,
  HistoryEvent,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  EventDiffs,
  History,
  HistoryDiff as LoadDiff,
  MarkKnownGood,
  RestoreKnownGood,
  RevertHistoryItem,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { selectedPair } from './historyDiff.ts'
import { pushMissingToast, revertHistoryEvent } from './historyRevert.ts'
import { useProfiles } from './store.ts'

async function diffsOf(game: string, profileId: string, events: HistoryEvent[]) {
  const diffs = await EventDiffs(
    game,
    profileId,
    events.map((ev) => ev.id),
  )
  const out: Record<string, HistoryDiff['items']> = {}
  for (const ev of events) {
    out[ev.id] = diffs?.[ev.id]?.items ?? []
  }
  return out
}

// Loads the changes of shown events that don't have them yet: a later page, or events a revert made.
function useShownDiffs({
  game,
  profileId,
  shown,
  items,
  setItems,
}: {
  game: string
  profileId: string
  shown: HistoryEvent[]
  items: Record<string, HistoryDiff['items']>
  setItems: Dispatch<SetStateAction<Record<string, HistoryDiff['items']>>>
}) {
  useEffect(() => {
    const missing = shown.filter((ev) => !(ev.id in items))
    if (game === '' || missing.length === 0) {
      return
    }
    let live = true
    diffsOf(game, profileId, missing)
      .then((more) => {
        if (live) {
          setItems((had) => ({ ...had, ...more }))
        }
      })
      .catch(reportUnexpected)
    return () => {
      live = false
    }
  }, [game, profileId, shown, items, setItems])
}

// The comparison of the two picked entries, null unless exactly two are picked.
function usePickedPair(game: string, profileId: string, picked: string[]) {
  const ids = selectedPair(picked)
  return useLoaded<HistoryDiff | null>(
    ids && game ? () => LoadDiff(game, profileId, ids[0], ids[1]) : null,
    [picked, game, profileId],
    null,
    reportUnexpected,
  ).data
}

/**
 * toastMissing is for a caller without the panel's own missing-mods UI: a revert that lacks mods says so in a toast.
 * pageSize, when set, shows that many events at first and loads each one's changes only once it's shown.
 */
export function useHistoryPanel(
  profileId: string,
  open: boolean,
  toastMissing = false,
  pageSize = Number.POSITIVE_INFINITY,
) {
  const game = useProfiles((s) => s.game)
  const [events, setEvents] = useState<HistoryEvent[]>([])
  const [items, setItems] = useState<Record<string, HistoryDiff['items']>>({})
  const [shown, setShown] = useState(pageSize)
  const [picked, setPicked] = useState<string[]>([])
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const [missingEvent, setMissingEvent] = useState('')
  const [missingNames, setMissingNames] = useState<string[]>([])
  const [missingWants, setMissingWants] = useState<
    Awaited<ReturnType<typeof revertHistoryEvent>>['missingWants']
  >([])
  useEffect(() => {
    if (!(open && game)) {
      return
    }
    let live = true
    setError('')
    setMissingEvent('')
    setMissingNames([])
    setMissingWants([])
    setPicked([])
    setShown(pageSize)
    History(game.id, profileId)
      .then(async (list) => {
        const rows = list ?? []
        const next = await diffsOf(game.id, profileId, rows.slice(0, pageSize))
        // Events and their changes arrive together, so the timeline draws once.
        if (live) {
          setEvents(rows)
          setItems(next)
        }
      })
      .catch(reportUnexpected)
    return () => {
      live = false
    }
  }, [open, game, profileId, pageSize])
  useShownDiffs({
    game: open ? (game?.id ?? '') : '',
    profileId,
    shown: events.slice(0, shown),
    items,
    setItems,
  })
  const pair = usePickedPair(game?.id ?? '', profileId, picked)
  return {
    game,
    events,
    items,
    shown,
    showMore: () => setShown((n) => n + pageSize),
    picked,
    pair,
    busy,
    error,
    missingEvent,
    missingNames,
    missingWants,
    setPicked,
    revertTo: async (id: string) => {
      if (!game) {
        return
      }
      setBusy(id)
      const result = await revertHistoryEvent(game.id, profileId, id, events)
      setEvents(result.events)
      setError(result.error)
      setMissingEvent(result.missingEvent)
      setMissingNames(result.missingNames)
      setMissingWants(result.missingWants)
      setBusy('')
      if (toastMissing) {
        pushMissingToast(result)
      }
    },
    revertItem: async (eventId: string, mod: string) => {
      if (!game) {
        return
      }
      setBusy(eventId)
      try {
        useProfiles.getState().replace(await RevertHistoryItem(game.id, profileId, eventId, mod))
        setEvents((await History(game.id, profileId)) ?? [])
      } catch (e) {
        setError(errorMessage(e))
      }
      setBusy('')
    },
    markGood: () => {
      if (!game) {
        return
      }
      setBusy('good')
      MarkKnownGood(game.id, profileId)
        .then(async () => setEvents((await History(game.id, profileId)) ?? []))
        .catch(reportUnexpected)
        .finally(() => setBusy(''))
    },
    restoreGood: () => {
      if (!game) {
        return
      }
      setBusy('restore')
      RestoreKnownGood(game.id, profileId)
        .then((next) => {
          useProfiles.getState().replace(next)
          return History(game.id, profileId)
        })
        .then((list) => setEvents(list ?? []))
        .catch(reportUnexpected)
        .finally(() => setBusy(''))
    },
  }
}
