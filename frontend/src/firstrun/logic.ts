export interface SetupState {
  installed: boolean
  smapiInstalled: boolean
  profileCount: number
}

// First run is for a game that is not found, or a first launch with no SMAPI and no profile.
// A broken SMAPI later is the loader banner's job, and no profiles is the detail pane's empty state.
export const shouldShowFirstRun = (s: SetupState): boolean =>
  !s.installed || (!s.smapiInstalled && s.profileCount === 0)
