export interface SetupState {
  installed: boolean
  loaderInstalled: boolean
  profileCount: number
}

// First run is for a game that is not found, or a first launch with no loader and no profile.
// A broken loader later is the loader banner's job, and no profiles is the detail pane's empty state.
export const shouldShowFirstRun = (s: SetupState): boolean =>
  !s.installed || (!s.loaderInstalled && s.profileCount === 0)

export type ChipState = 'done' | 'current' | 'todo'

// A loader install that failed and was skipped leaves its step open, not ticked.
export const loaderChip = (step: number, loaderStep: number, installed: boolean): ChipState => {
  if (step === loaderStep) {
    return 'current'
  }
  return step > loaderStep && installed ? 'done' : 'todo'
}
