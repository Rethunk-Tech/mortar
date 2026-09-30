import { msg } from '@lingui/core/macro'
import { Events, Window } from '@wailsio/runtime'
import { create } from 'zustand'
import { ModName } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'
import type {
  Arrival,
  Rejection,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/nxmsvc/models.ts'
import {
  Assign,
  Ignore,
  Inbox,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/nxmsvc/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { SendNotification } from '../../bindings/github.com/wailsapp/wails/v3/pkg/services/notifications/notificationservice.ts'
import { i18n } from '../i18n/index.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { directProfile, NXM_GAME } from './route.ts'

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

// A desktop notification stands in for the on-screen prompt while the window is minimised; clicking it brings the
// window up.
async function notify(arrival: Arrival, title: string, body: string) {
  if (!(await Window.IsMinimised())) {
    return
  }
  await SendNotification({ id: `nxm-${arrival.id}`, title, body })
}

const names = new Map<number, Promise<string>>()

async function install(arrival: Arrival, profile: Profile) {
  try {
    await Assign(arrival.id, NXM_GAME, profile.id)
  } catch (e) {
    // Nothing downloads, so the prompt keeps the link for another profile to take.
    useToasts.getState().push({
      kind: 'error',
      title: i18n._(msg`Could not start the Nexus download`),
      body: errorMessage(e),
    })
    useNxm.getState().add(arrival)
    return
  }
  const name = await modName(arrival.link.modId)
  const profileName = profile.name
  const title = i18n._(msg`Downloading ${name} into ${profileName}`)
  useToasts.getState().push({ kind: 'info', title })
  await notify(arrival, title, '')
}

export const fallbackName = (modId: number) => i18n._(msg`Nexus mod ${modId}`)

// The mod's Nexus page title, fetched once per mod; the fallback stands in when signed out or offline.
export function modName(modId: number): Promise<string> {
  let name = names.get(modId)
  if (!name) {
    name = ModName(modId)
      .then((n) => n || fallbackName(modId))
      .catch(() => {
        names.delete(modId)
        return fallbackName(modId)
      })
    names.set(modId, name)
  }
  return name
}

export const useNxm = create<{
  arrivals: Arrival[]
  add: (arrival: Arrival) => void
  // Rejects when Mortar refuses the download; the arrival stays for the prompt to show why.
  choose: (id: number, profile: string) => Promise<void>
  dismiss: (id: number) => void
}>((set) => ({
  arrivals: [],
  add: (arrival) =>
    set((s) =>
      s.arrivals.some((a) => a.id === arrival.id) ? s : { arrivals: [...s.arrivals, arrival] },
    ),
  choose: async (id, profile) => {
    await Assign(id, NXM_GAME, profile)
    set((s) => ({ arrivals: s.arrivals.filter((a) => a.id !== id) }))
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
    const { game, openId, profiles } = useProfiles.getState()
    const direct = directProfile(useNav.getState().route, game?.id, openId, profiles)
    if (direct) {
      install(a, direct).catch(reportUnexpected)
      return
    }
    useNxm.getState().add(a)
    modName(a.link.modId)
      .then((name) =>
        notify(
          a,
          i18n._(msg`Mortar · Install ${name}?`),
          i18n._(
            msg`You started this download on Nexus. Show Mortar to choose the profile it goes into.`,
          ),
        ),
      )
      .catch(reportUnexpected)
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
