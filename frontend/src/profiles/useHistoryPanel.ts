import { useEffect, useState } from 'react'
import type {
  HistoryDiff,
  HistoryEvent,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  EventDiff,
  History,
  HistoryDiff as LoadDiff,
  MarkKnownGood,
  RestoreKnownGood,
  RevertHistoryItem,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { selectedPair } from './historyDiff.ts'
import { pushMissingToast, revertHistoryEvent } from './historyRevert.ts'
import { useProfiles } from './store.ts'

/** toastMissing is for a caller without the panel's own missing-mods UI: a revert that lacks mods says so in a toast. */
export function useHistoryPanel(profileId: string, open: boolean, toastMissing = false) {
  const game = useProfiles((s) => s.game)
  const [events, setEvents] = useState<HistoryEvent[]>([])
  const [items, setItems] = useState<Record<string, HistoryDiff['items']>>({})
  const [picked, setPicked] = useState<string[]>([])
  const [pair, setPair] = useState<HistoryDiff | null>(null)
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
    setPair(null)
    History(game.id, profileId)
      .then(async (list) => {
        const rows = list ?? []
        if (live) {
          setEvents(rows)
        }
        const next: Record<string, HistoryDiff['items']> = {}
        await Promise.all(
          rows.map(async (ev) => {
            next[ev.id] = (await EventDiff(game.id, profileId, ev.id))?.items ?? []
          }),
        )
        if (live) {
          setItems(next)
        }
      })
      .catch(reportUnexpected)
    return () => {
      live = false
    }
  }, [open, game, profileId])
  useEffect(() => {
    const ids = selectedPair(picked)
    if (!(ids && game)) {
      setPair(null)
      return
    }
    LoadDiff(game.id, profileId, ids[0], ids[1]).then(setPair).catch(reportUnexpected)
  }, [picked, game, profileId])
  return {
    game,
    events,
    items,
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
