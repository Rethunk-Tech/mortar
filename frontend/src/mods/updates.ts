import { msg, plural } from '@lingui/core/macro'
import { create } from 'zustand'
import type { UpdatesResult } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import {
  CheckUpdatesNow,
  Updates,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import { useTab } from '../game/tab.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useBadges } from './badges.ts'
import { updateCount } from './lookup.ts'
import { useNexusDetails } from './nexusDetails.ts'
import { openTarget } from './storeView.ts'

function checkedLabel(at: number | null, now: number, unknown: boolean): string {
  if (unknown) {
    return i18n._(msg`Some mods could not be checked, so more updates may show up later.`)
  }
  if (at === null) {
    return i18n._(msg`Checked for updates`)
  }
  const when = formatWhen(at, { now })
  return i18n._(msg`Checked for updates · ${when}`)
}

const MS_PER_HOUR = 3_600_000
let hourlyTimer: ReturnType<typeof setInterval> | undefined
const inFlight = new Map<string, Promise<UpdatesResult>>()

const countFor = (updates: UpdatesResult, id: string) =>
  updateCount(
    updates,
    useProfiles.getState().profiles.find((p) => p.id === id),
    useNexusDetails.getState().byId,
  )

function syncBadge() {
  const { openId } = useProfiles.getState()
  const { updates } = useUpdates.getState()
  if (!(openId && updates)) {
    return
  }
  useBadges.getState().patch(openId, { updates: countFor(updates, openId) })
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
      const count = countFor(updates, at.id)
      if (count > 0 && useSettings.getState().notifyModUpdates) {
        useToasts.getState().push({
          kind: 'info',
          title: i18n._(
            msg`${plural(count, { one: '# mod update available', other: '# mod updates available' })}`,
          ),
          action: {
            label: i18n._(msg`Review`),
            run: () => {
              useTab.getState().setTab('mods')
              useUpdates.getState().setReviewing(true)
            },
          },
        })
      }
    } catch (e) {
      toastError(i18n._(msg`Could not check for mod updates`), e, {
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
    return countFor(updates, at.id)
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
// The review is of the open profile's updates, so it closes when another profile opens. The Mods tab cannot do this:
// it mounts after the chip or toast that opens the review has already asked for it.
let reviewedProfile = useProfiles.getState().openId
useProfiles.subscribe((s) => {
  if (s.openId !== reviewedProfile) {
    reviewedProfile = s.openId
    useUpdates.setState({ reviewing: false })
  }
})
useSettings.subscribe(syncHourlyRecheck)
useNexusDetails.subscribe(syncBadge)
syncHourlyRecheck()

export { checkedLabel, loadUpdates, useUpdates }
