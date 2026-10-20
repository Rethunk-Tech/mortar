import { create } from 'zustand'
import { SetTipsSeen } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useSettings } from '../settings/store.ts'

// Sites that publish whatever is uploaded, with no review and no malware scan.
const UNMODERATED = new Set(['github', 'itch'])

const seenKey = (source: string) => `source-warning:${source}`

export const useUnmoderatedAsk = create<{
  ask: { sources: string[]; answer: (go: boolean) => void } | null
}>(() => ({ ask: null }))

/** The unmoderated sites among these that the player has not been warned about yet. */
export function unwarnedSources(sources: readonly string[]): string[] {
  const seen = useSettings.getState().tipsSeen ?? []
  return [...new Set(sources)].filter((s) => UNMODERATED.has(s) && !seen.includes(seenKey(s)))
}

async function warnNow(sources: readonly string[]): Promise<boolean> {
  const fresh = unwarnedSources(sources)
  if (fresh.length === 0) {
    return true
  }
  const go = await new Promise<boolean>((answer) =>
    useUnmoderatedAsk.setState({ ask: { sources: fresh, answer } }),
  )
  useUnmoderatedAsk.setState({ ask: null })
  if (go) {
    await SetTipsSeen([...(useSettings.getState().tipsSeen ?? []), ...fresh.map(seenKey)])
  }
  return go
}

// One question at a time: a second download waits for the first answer, which may already cover its site.
let pending: Promise<unknown> = Promise.resolve()

/** Before the first download from an unmoderated site, says so once; resolves false when the player backs out. */
export function warnUnmoderated(sources: readonly string[]): Promise<boolean> {
  const next = pending.then(() => warnNow(sources))
  pending = next.catch(() => undefined)
  return next
}
