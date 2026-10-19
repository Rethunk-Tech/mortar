import type { Case } from './render-budget.ts'

const KB_PER_GB = 1024 * 1024
const HELD_GB = 3

const heldDownload = {
  items: [
    {
      id: 'render-large',
      kind: 'install',
      game: 'stardew',
      profileId: '',
      name: 'Render Large Pack',
      fileName: 'large.zip',
      state: 'confirm-large',
      sizeKb: HELD_GB * KB_PER_GB,
      progress: 0,
      speed: 0,
      error: '',
    },
  ],
  paused: false,
  limitedUntil: 0,
}

/** The large-download modal opens when the queue announces one download over 2 GB. */
export function largeDownloadCases(): Case[] {
  return [
    {
      name: 'large-download-modal',
      // The held queue arrives through the event the app listens to, so the figure is the dialog's, not a page load's.
      run: async (p) => {
        await p.evaluate((state) => {
          const w = globalThis as unknown as {
            _wails: { dispatchWailsEvent: (e: { name: string; data: unknown }) => void }
          }
          w._wails.dispatchWailsEvent({ name: 'queue:changed', data: state })
        }, heldDownload)
        await p.getByRole('dialog', { name: /very large download/ }).waitFor()
      },
    },
  ]
}
