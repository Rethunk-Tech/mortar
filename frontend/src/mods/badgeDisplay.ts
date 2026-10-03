export const SIDEBAR_BADGES_ALL = 'problemsAndUpdates'
export const SIDEBAR_BADGES_PROBLEMS = 'problems'
export const SIDEBAR_BADGES_OFF = 'off'

export function showUpdateBadge(mode: string): boolean {
  return mode === SIDEBAR_BADGES_ALL
}

export function showProblemBadge(mode: string): boolean {
  return mode === SIDEBAR_BADGES_ALL || mode === SIDEBAR_BADGES_PROBLEMS
}

export type HealthTone = 'red' | 'amber' | 'none'

export interface HealthLabels {
  missing: (n: number) => string
  problems: (n: number) => string
  updates: (n: number) => string
}

export function healthView(
  counts: { missing?: number; problems?: number; updates?: number } | undefined,
  mode: string,
  labels: HealthLabels,
): { tone: HealthTone; tooltip: string; value: number } {
  const missing = showProblemBadge(mode) ? (counts?.missing ?? 0) : 0
  const problems = showProblemBadge(mode) ? (counts?.problems ?? 0) : 0
  const updates = showUpdateBadge(mode) ? (counts?.updates ?? 0) : 0
  const lines = [
    missing > 0 ? labels.missing(missing) : '',
    problems > 0 ? labels.problems(problems) : '',
    updates > 0 ? labels.updates(updates) : '',
  ].filter((line) => line !== '')
  if (lines.length === 0) {
    return { tone: 'none', tooltip: '', value: 0 }
  }
  const tone: HealthTone = missing > 0 || problems > 0 ? 'red' : 'amber'
  return {
    tone,
    tooltip: lines.join('\n'),
    value: tone === 'red' ? missing + problems : updates,
  }
}

export function runBackgroundBadgeChecks(
  mode: string,
  backgroundBadgeChecks: boolean | null | undefined,
): boolean {
  if (mode === SIDEBAR_BADGES_OFF) {
    return false
  }
  return backgroundBadgeChecks !== false
}
