import type { ToastInput } from '../toasts/store.ts'

export function lastRunCrashToast(
  crashed: boolean,
  copy: { title: string; body: string; action: string },
  report: () => unknown,
): ToastInput | undefined {
  if (!crashed) {
    return undefined
  }
  return {
    kind: 'warning',
    title: copy.title,
    body: copy.body,
    action: { label: copy.action, run: report },
  }
}
