import { Events } from '@wailsio/runtime'
import { useMortarUpdate } from '../settings/updates.ts'

let listening = false

export function initMortarUpdateBackground() {
  if (listening || typeof Events.On !== 'function') {
    return
  }
  listening = true
  useMortarUpdate
    .getState()
    .load()
    .catch(() => undefined)
  Events.On('update:staged', (event) => {
    const release = event.data
    if (!release) {
      return
    }
    useMortarUpdate.setState({ release, phase: 'ready', error: '' })
  })
}
