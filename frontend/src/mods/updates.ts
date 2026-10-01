import { msg, plural } from '@lingui/core/macro'
import { create } from 'zustand'
import type { UpdatesResult } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { Updates } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import { i18n } from '../i18n/index.ts'
import { ago } from '../notes/ago.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useBadges } from './badges.ts'
import { updateCount } from './lookup.ts'

function checkedWhen(at: number | null, now: number) {
  if (at === null) {
    return null
  }
  return ago(now - at)
}

function checkedWithSmapi(at: number | null, now: number, unknown: boolean): string {
  if (unknown) {
    return i18n._(msg`Some mods could not be checked, so more updates may show up later.`)
  }
  if (at === null) {
    return i18n._(msg`Checked with SMAPI's update service`)
  }
  const { unit, n } = ago(now - at)
  if (unit === 'now') {
    return i18n._(msg`Checked with SMAPI's update service · just now`)
  }
  if (unit === 'minute') {
    return i18n._(
      msg`Checked with SMAPI's update service · ${plural(n, { one: '# minute ago', other: '# minutes ago' })}`,
    )
  }
  if (unit === 'hour') {
    return i18n._(
      msg`Checked with SMAPI's update service · ${plural(n, { one: '# hour ago', other: '# hours ago' })}`,
    )
  }
  return i18n._(
    msg`Checked with SMAPI's update service · ${plural(n, { one: '# day ago', other: '# days ago' })}`,
  )
}

const MS_PER_HOUR = 3_600_000
let hourlyTimer: ReturnType<typeof setInterval> | undefined

function ignoreRecheckError() {
  return
}

const useUpdates = create<{
  updates: UpdatesResult | null
  checkedAt: number | null
  reviewing: boolean
  load: () => Promise<void>
  setReviewing: (reviewing: boolean) => void
}>((set) => ({
  updates: null,
  checkedAt: null,
  reviewing: false,
  load: async () => {
    const { game, openId } = useProfiles.getState()
    if (!(game && openId)) {
      return
    }
    try {
      const updates = await Updates(game.id, openId)
      if (useProfiles.getState().openId === openId) {
        set({ updates, checkedAt: Date.now() })
      }
      const profile = useProfiles.getState().profiles.find((p) => p.id === openId)
      useBadges.getState().patch(openId, { updates: updateCount(updates, profile) })
    } catch (e) {
      useToasts.getState().push({
        kind: 'error',
        title: i18n._(msg`Couldn't reach Nexus`),
        body: errorMessage(e),
        action: {
          label: i18n._(msg`Retry now`),
          run: () => useUpdates.getState().load(),
        },
      })
    }
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
  hourlyTimer = setInterval(() => {
    useUpdates.getState().load().catch(ignoreRecheckError)
  }, MS_PER_HOUR)
}

useProfiles.subscribe(syncHourlyRecheck)
syncHourlyRecheck()

export { checkedWhen, checkedWithSmapi, useUpdates }
