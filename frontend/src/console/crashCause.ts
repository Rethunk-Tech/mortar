export type CrashCauseKind = 'missing-file' | 'asset-load' | 'mod-exception'

export function crashCauseKind(reason: string): CrashCauseKind | '' {
  switch (reason) {
    case 'missing-file':
    case 'asset-load':
    case 'mod-exception':
      return reason
    default:
      return ''
  }
}

export function crashCauseDetailLine(detail: string): string {
  return detail.split('\n')[0] ?? ''
}
