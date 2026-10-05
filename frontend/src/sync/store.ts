import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type {
  Offer,
  Stall,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/syncsvc/models.ts'
import {
  Offers,
  Stalled,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/syncsvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

const useSync = create<{
  offers: Offer[]
  stalled: Stall[]
  open: boolean
  setOpen: (open: boolean) => void
}>((set) => ({
  offers: [],
  stalled: [],
  open: false,
  setOpen: (open) => set({ open }),
}))

const offerKey = (o: Pick<Offer, 'game' | 'remote'>) => `${o.game}/${o.remote}`

// Each change is announced once, by a notification that opens the dialog where it is answered.
function present(offers: Offer[]) {
  const before = new Set(useSync.getState().offers.map(offerKey))
  useSync.setState({ offers })
  for (const o of offers.filter((x) => !before.has(offerKey(x)))) {
    const title = o.conflict
      ? i18n._(msg`${o.name} was changed on ${o.machine} and here`)
      : i18n._(msg`${o.name} was changed on ${o.machine}`)
    useToasts.getState().push({
      kind: 'info',
      title,
      action: { label: i18n._(msg`Review`), run: () => useSync.getState().setOpen(true) },
    })
  }
}

// A profile waiting on another machine's file is announced once, so a sync that stopped is never silent.
function presentStalled(stalled: Stall[]) {
  const before = new Set(useSync.getState().stalled.map(offerKey))
  useSync.setState({ stalled })
  for (const st of stalled.filter((x) => !before.has(offerKey(x)))) {
    useToasts.getState().push({
      kind: 'info',
      title: i18n._(msg`${st.name} is not syncing yet`),
      body: i18n._(msg`Waiting for ${st.machine}'s profile file to finish syncing`),
      action: { label: i18n._(msg`Review`), run: () => useSync.getState().setOpen(true) },
    })
  }
}

function initSync() {
  Events.On('sync:stalled', (event) => presentStalled((event.data as Stall[] | null) ?? []))
  Stalled()
    .then((stalled) => presentStalled(stalled ?? []))
    .catch(reportUnexpected)
  Events.On('sync:offers', (event) => present((event.data as Offer[] | null) ?? []))
  Offers()
    .then((offers) => present(offers ?? []))
    .catch(reportUnexpected)
}

export { initSync, useSync }
