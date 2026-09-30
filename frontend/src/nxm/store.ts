import { msg } from '@lingui/core/macro'
import { Events, Window } from '@wailsio/runtime'
import { create } from 'zustand'
import type {
  Arrival,
  Rejection,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/nxmsvc/models.ts'
import {
  Assign,
  Ignore,
  Inbox,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/nxmsvc/service.ts'
import { SendNotification } from '../../bindings/github.com/wailsapp/wails/v3/pkg/services/notifications/notificationservice.ts'
import { i18n } from '../i18n/index.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

const rejectionText = (reason: string): string => {
  switch (reason) {
    case 'game':
      return i18n._(msg`Mortar takes Stardew Valley links only.`)
    case 'expired':
      return i18n._(msg`The link has expired. Click Mod Manager Download on Nexus again.`)
    case 'user':
      return i18n._(msg`The link was made for another Nexus account than the one signed in.`)
    case 'signedIn':
      return i18n._(msg`Sign in to Nexus Mods in Settings before downloading from a link.`)
    default:
      return i18n._(msg`This is not a Mod Manager Download link from Nexus.`)
  }
}

// A desktop notification stands in for the card while the window is minimised; clicking it brings the window up.
async function notifyArrival(arrival: Arrival) {
  if (!(await Window.IsMinimised())) {
    return
  }
  await SendNotification({
    id: `nxm-${arrival.id}`,
    title: i18n._(msg`Mortar · Install Nexus mod ${arrival.link.modId}?`),
    body: i18n._(
      msg`You started this download on Nexus. Show Mortar to choose the profile it goes into.`,
    ),
  })
}

// The game an nxm link for stardewvalley belongs to.
export const NXM_GAME = 'stardew'

export const useNxm = create<{
  arrivals: Arrival[]
  add: (arrival: Arrival) => void
  choose: (id: number, profile: string) => void
  dismiss: (id: number) => void
}>((set) => ({
  arrivals: [],
  add: (arrival) =>
    set((s) =>
      s.arrivals.some((a) => a.id === arrival.id) ? s : { arrivals: [...s.arrivals, arrival] },
    ),
  choose: (id, profile) => {
    set((s) => ({ arrivals: s.arrivals.filter((a) => a.id !== id) }))
    Assign(id, NXM_GAME, profile).catch(reportUnexpected)
  },
  dismiss: (id) => {
    set((s) => ({ arrivals: s.arrivals.filter((a) => a.id !== id) }))
    Ignore(id).catch(reportUnexpected)
  },
}))

export async function initNxm(): Promise<void> {
  const seen = new Set<number>()
  const reject = (r: Rejection) => {
    if (!seen.has(r.id)) {
      seen.add(r.id)
      useToasts.getState().push({
        kind: 'error',
        title: i18n._(msg`Nexus link not used`),
        body: rejectionText(r.reason),
      })
    }
  }
  const arrive = (a: Arrival) => {
    useNxm.getState().add(a)
    notifyArrival(a).catch(reportUnexpected)
  }
  Events.On('nxm:rejected', (e) => reject(e.data))
  Events.On('nxm:arrived', (e) => arrive(e.data))
  const inbox = await Inbox()
  for (const r of inbox.rejections ?? []) {
    reject(r)
  }
  for (const a of inbox.arrivals ?? []) {
    arrive(a)
  }
}
