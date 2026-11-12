import { Events } from '@wailsio/runtime'
import { useMortarUpdate } from '../settings/updates.ts'

export function initMortarUpdateBackground() {
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
