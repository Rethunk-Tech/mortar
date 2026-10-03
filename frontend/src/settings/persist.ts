import { errorText } from '../toasts/report.ts'
import type { useToasts } from '../toasts/store.ts'

export function persist(
  run: () => Promise<void>,
  push: ReturnType<typeof useToasts.getState>['push'],
  title: string,
) {
  run().catch((err: unknown) => {
    const body = errorText(err)
    push({ kind: 'error', title, ...(body ? { body } : {}) })
  })
}
