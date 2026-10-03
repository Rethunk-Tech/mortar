export function canBisectCrash(crash: { cause?: unknown }): boolean {
  return crash.cause === null || crash.cause === undefined
}
