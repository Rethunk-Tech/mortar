import { Clipboard } from '@wailsio/runtime'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

/** Copies text, toasts `done` and then calls `onCopied`; a failure is reported and skips both. */
export function copyText(text: string, done: string, onCopied?: () => unknown): void {
  Clipboard.SetText(text).then(() => {
    useToasts.getState().push({ kind: 'success', title: done })
    return onCopied?.()
  }, reportUnexpected)
}
