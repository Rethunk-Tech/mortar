export function runExitText(
  exit: { code?: number; signal?: string; stopped?: boolean } | null | undefined,
): string {
  if (exit === null || exit === undefined) {
    return ''
  }
  if (exit.stopped) {
    return 'stopped'
  }
  const code = exit.code ?? 0
  const signal = exit.signal ?? ''
  if (signal !== '') {
    if (code !== 0) {
      return `exited with code ${code} (${signal})`
    }
    return `exited with ${signal}`
  }
  if (code !== 0) {
    return `exited with code ${code}`
  }
  return 'exited with code 0'
}
