import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type { UpdatesResult } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import {
  CheckUpdatesNow,
  Updates,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { errorDetails } from '../toasts/errorKind.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useBadges } from './badges.ts'
import { updateCount } from './lookup.ts'
import { useNexusDetails } from './nexusDetails.ts'
import { openTarget } from './storeView.ts'

function checkedWithSmapi(at: number | null, now: number, unknown: boolean): string {
  if (unknown) {
    return i18n._(msg`Some mods could not be checked, so more updates may show up later.`)
  }
  if (at === null) {
    return i18n._(msg`Checked with SMAPI's update service`)
  }
  const when = formatWhen(at, { now })
  return i18n._(msg`Checked with SMAPI's update service · ${when}`)
}

const MS_PER_HOUR = 3_600_000
let hourlyTimer: ReturnType<typeof setInterval> | undefined
const inFlight = new Map<string, Promise<UpdatesResult>>()

function syncBadge() {
  const { openId, profiles } = useProfiles.getState()
  const { updates } = useUpdates.getState()
  if (!(openId && updates)) {
    return
  }
  const profile = profiles.find((p) => p.id === openId)
  useBadges
    .getState()
    .patch(openId, { updates: updateCount(updates, profile, useNexusDetails.getState().byId) })
}

function loadUpdates(game: string, profile: string): Promise<UpdatesResult> {
  const key = `${game}/${profile}`
  const existing = inFlight.get(key)
  if (existing !== undefined) {
    return existing
  }
  const promise = Updates(game, profile)
  inFlight.set(key, promise)
  promise.then(
    () => {
      if (inFlight.get(key) === promise) {
        inFlight.delete(key)
      }
    },
    () => {
      if (inFlight.get(key) === promise) {
        inFlight.delete(key)
      }
    },
  )
  return promise
}

function ignoreRecheckError() {
  return
}

const useUpdates = create<{
  updates: UpdatesResult | null
  checkedAt: number | null
  reviewing: boolean
  load: () => Promise<void>
  checkNow: () => Promise<number | null>
  setReviewing: (reviewing: boolean) => void
}>((set) => ({
  updates: null,
  checkedAt: null,
  reviewing: false,
  load: async () => {
    const at = openTarget()
    if (!at) {
      return
    }
    try {
      const updates = await loadUpdates(at.game, at.id)
      if (useProfiles.getState().openId === at.id) {
        set({ updates, checkedAt: Date.now() })
      }
      syncBadge()
      const count = updateCount(
        updates,
        useProfiles.getState().profiles.find((p) => p.id === at.id),
      )
      if (count > 0 && useSettings.getState().notifyModUpdates) {
        useToasts.getState().push({
          kind: 'info',
          title: i18n._(msg`${count} updates available`),
        })
      }
    } catch (e) {
      useToasts.getState().push({
        kind: 'error',
        title: i18n._(msg`SMAPI update check failed`),
        body: errorMessage(e),
        detail: errorDetails(e),
        action: {
          label: i18n._(msg`Retry now`),
          run: () => useUpdates.getState().load(),
        },
      })
    }
  },
  // An explicit check: asks SMAPI's API again, ignoring cached answers, and returns how many updates it found.
  checkNow: async () => {
    const at = openTarget()
    if (!at) {
      return null
    }
    const updates = await CheckUpdatesNow(at.game, at.id)
    if (useProfiles.getState().openId === at.id) {
      set({ updates, checkedAt: Date.now() })
    }
    syncBadge()
    return updateCount(
      updates,
      useProfiles.getState().profiles.find((p) => p.id === at.id),
    )
  },
  setReviewing: (reviewing) => set({ reviewing }),
}))

function syncHourlyRecheck() {
  const { game, openId } = useProfiles.getState()
  if (hourlyTimer !== undefined) {
    clearInterval(hourlyTimer)
    hourlyTimer = undefined
  }
  if (!(game && openId)) {
    return
  }
  const minutes = useSettings.getState().updateCheckIntervalMinutes || 60
  hourlyTimer = setInterval(
    () => {
      useUpdates.getState().load().catch(ignoreRecheckError)
    },
    minutes * (MS_PER_HOUR / 60),
  )
}

useProfiles.subscribe(syncHourlyRecheck)
useSettings.subscribe(syncHourlyRecheck)
useNexusDetails.subscribe(syncBadge)
syncHourlyRecheck()

export { checkedWithSmapi, loadUpdates, useUpdates }
