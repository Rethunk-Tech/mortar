import { Events } from '@wailsio/runtime'
import { ConfirmQuit } from '../bindings/github.com/Rethunk-AI/mortar/quitservice.ts'
import { useQueue } from './queue/store.ts'

export function initQuit() {
  Events.On('quit:requested', (event) => {
    const summary = String(event.data ?? '')
    if (summary && !window.confirm(summary)) {
      return
    }
    ConfirmQuit().catch(() => undefined)
  })
  Events.On('queue:open', () => useQueue.getState().setOpen(true))
}
