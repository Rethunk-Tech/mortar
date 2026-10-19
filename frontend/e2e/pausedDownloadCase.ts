import type { Case } from './render-budget.ts'

const KB_PER_GB = 1024 * 1024
const GB = 1024 ** 3

const item = (id: string, name: string, state: string, over: object = {}) => ({
  id,
  kind: 'install',
  game: 'stardew',
  profileId: '',
  name,
  fileName: `${id}.zip`,
  state,
  sizeKb: 4.9 * KB_PER_GB,
  progress: 0,
  speed: 0,
  error: '',
  ...over,
})

const queue = {
  items: [
    item('render-down', 'Render Active Pack', 'downloading', {
      progress: 40,
      speed: 6_000_000,
    }),
    item('render-paused', 'Render Paused Pack', 'paused', { progress: 24, partial: 1.2 * GB }),
    item('render-yield', 'Render Yielded Pack', 'paused', {
      progress: 10,
      partial: 0.5 * GB,
      autoResume: true,
    }),
    item('render-failed', 'Render Failed Pack', 'failed', {
      error: '[network] timeout',
      errorKind: 'network',
      partial: 2 * GB,
    }),
    item('render-next', 'Render Waiting Pack', 'queued'),
  ],
  paused: false,
  limitedUntil: 0,
}

/** A paused download shows how much is on disk and a Resume button; a failed one with a partial offers Resume. */
export function pausedDownloadCases(): Case[] {
  return [
    {
      name: 'paused-download-sheet',
      // The failed row's toast leaves once, inside the idle watch.
      budget: { idleUpdates: 10, idleCommits: 1 },
      run: async (p) => {
        await p.evaluate((state) => {
          const w = globalThis as unknown as {
            _wails: { dispatchWailsEvent: (e: { name: string; data: unknown }) => void }
          }
          w._wails.dispatchWailsEvent({ name: 'queue:changed', data: state })
        }, queue)
        await p.getByRole('button', { name: /^Downloads:/ }).click()
        await p.getByText(/Paused at 1\.2 GB of 4\.9 GB/).waitFor()
      },
    },
  ]
}
