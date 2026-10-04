import type { Usage as DiskUse } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/models.ts'

const PROGRESS_MS = 80

export function beginUsageLoad(opts: {
  usage: () => Promise<DiskUse>
  progress: () => Promise<{ measuring: boolean; bytes: number }>
  setBytes: (n: number) => void
  setUsage: (u: DiskUse) => void
  onError: (e: unknown) => void
  ms?: number
  every?: (fn: () => void, ms: number) => ReturnType<typeof setInterval>
  stopEvery?: (id: ReturnType<typeof setInterval>) => void
}): () => void {
  let live = true
  const every = opts.every ?? ((fn, ms) => globalThis.setInterval(fn, ms))
  const stopEvery = opts.stopEvery ?? ((id) => globalThis.clearInterval(id))
  const tick = every(() => {
    opts
      .progress()
      .then((p) => {
        if (live && p.measuring) {
          opts.setBytes(p.bytes)
        }
      })
      .catch(() => undefined)
  }, opts.ms ?? PROGRESS_MS)
  opts
    .usage()
    .then((u) => {
      if (live) {
        opts.setUsage(u)
      }
    })
    .catch((e: unknown) => {
      if (live) {
        opts.onError(e)
      }
    })
    .finally(() => {
      stopEvery(tick)
    })
  return () => {
    live = false
    stopEvery(tick)
  }
}
