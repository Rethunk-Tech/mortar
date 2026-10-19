import type { Case } from './render-budget.ts'

// Queue.State's binding id.
const queueStateId = '1504184018'
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

/** The large-download modal opens when the page loads a queue holding one download over 2 GB. */
export function largeDownloadCases(): Case[] {
  return [
    {
      name: 'large-download-modal',
      // The case reloads the whole app to load the held queue, so its renders are a page load's; the dialog itself is ~25.
      budget: { updates: 2500 },
      setup: (p) =>
        p.route('**/wails/runtime', async (route) => {
          if ((route.request().postData() ?? '').includes(queueStateId)) {
            await route.fulfill({
              status: 200,
              contentType: 'application/json',
              body: JSON.stringify(heldDownload),
            })
            return
          }
          await route.continue()
        }),
      run: async (p) => {
        await p.reload()
        await p.getByRole('dialog', { name: /very large download/ }).waitFor()
      },
    },
  ]
}
