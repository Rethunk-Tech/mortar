export interface SetupState {
  installed: boolean
  profileCount: number
}

// Setup runs for a game that is not found, and stays until the game has a profile: the game's screen needs one.
// A broken loader later is the loader banner's job.
export const shouldShowFirstRun = (s: SetupState): boolean => !s.installed || s.profileCount === 0

export type ChipState = 'done' | 'current' | 'todo'

// A loader install that failed and was skipped leaves its step open, not ticked.
export const loaderChip = (step: number, loaderStep: number, installed: boolean): ChipState => {
  if (step === loaderStep) {
    return 'current'
  }
  return step > loaderStep && installed ? 'done' : 'todo'
}

// Setup installs a loader only when the game has one that is not part of the game itself.
export const needsLoaderStep = (
  loaderId: string | undefined,
  loaders: readonly { id: string; builtin: boolean }[] | null | undefined,
): boolean => Boolean(loaderId) && !(loaders ?? []).find((l) => l.id === loaderId)?.builtin
