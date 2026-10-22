export function crashCauseDetailLine(detail: string): string {
  return detail.split('\n')[0] ?? ''
}
