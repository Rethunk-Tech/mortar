import { Clipboard } from '@wailsio/runtime'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

export function copyText(text: string, done: string) {
  Clipboard.SetText(text).then(
    () => useToasts.getState().push({ kind: 'success', title: done }),
    reportUnexpected,
  )
}
