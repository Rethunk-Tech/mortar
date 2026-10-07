import { Clipboard } from '@wailsio/runtime'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

/** Resolves true once the text is on the clipboard; a failure is reported and resolves false. */
export function copyText(text: string, done: string): Promise<boolean> {
  return Clipboard.SetText(text).then(
    () => {
      useToasts.getState().push({ kind: 'success', title: done })
      return true
    },
    (err: unknown) => {
      reportUnexpected(err)
      return false
    },
  )
}
