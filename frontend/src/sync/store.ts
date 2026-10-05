import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type { Offer } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/syncsvc/models.ts'
import { Offers } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/syncsvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

export const useSync = create<{
  offers: Offer[]
  open: boolean
  setOpen: (open: boolean) => void
}>((set) => ({
  offers: [],
  open: false,
  setOpen: (open) => set({ open }),
}))

const offerKey = (o: Offer) => `${o.game}/${o.remote}`

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

export function initSync() {
  Events.On('sync:offers', (event) => present((event.data as Offer[] | null) ?? []))
  Offers()
    .then((offers) => present(offers ?? []))
    .catch(reportUnexpected)
}
