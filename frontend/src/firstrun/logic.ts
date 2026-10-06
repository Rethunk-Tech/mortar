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
