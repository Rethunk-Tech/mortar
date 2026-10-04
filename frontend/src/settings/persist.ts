import { reportError } from '../toasts/report.ts'
import type { useToasts } from '../toasts/store.ts'

export function persist(
  run: () => Promise<void>,
  _push: ReturnType<typeof useToasts.getState>['push'],
  title: string,
) {
  run().catch(reportError(title))
}
