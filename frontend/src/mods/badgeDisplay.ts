export const SIDEBAR_BADGES_ALL = 'problemsAndUpdates'
export const SIDEBAR_BADGES_PROBLEMS = 'problems'
export const SIDEBAR_BADGES_OFF = 'off'

export function showUpdateBadge(mode: string): boolean {
  return mode === SIDEBAR_BADGES_ALL
}

export function showProblemBadge(mode: string): boolean {
  return mode === SIDEBAR_BADGES_ALL || mode === SIDEBAR_BADGES_PROBLEMS
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
