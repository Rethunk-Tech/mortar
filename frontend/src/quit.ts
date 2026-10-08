import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import { ConfirmQuit, PendingQuit } from '../bindings/github.com/Rethunk-Tech/mortar/quitservice.ts'
import { useQueue } from './queue/store.ts'

interface QuitState {
  message: string
  resolve: ((confirmed: boolean) => void) | null
}
const useQuitPrompt = create<QuitState>(() => ({ message: '', resolve: null }))

function askQuit(message: string): Promise<boolean> {
  return new Promise((resolve) => useQuitPrompt.setState({ message, resolve }))
}

async function handleQuit(event: { data: unknown }) {
  const summary = String(event.data ?? '')
  if (summary && !(await askQuit(summary))) {
    return
  }
  await ConfirmQuit()
}

function initQuit() {
  Events.On('quit:requested', (event) => {
    handleQuit(event).catch(() => undefined)
  })
  // A quit asked while the window was closed to the tray reaches the page built to answer it here.
  PendingQuit()
    .then((summary) => (summary ? handleQuit({ data: summary }) : undefined))
    .catch(() => undefined)
  Events.On('queue:open', () => useQueue.getState().setOpen(true))
}

export { askQuit, initQuit, useQuitPrompt }
