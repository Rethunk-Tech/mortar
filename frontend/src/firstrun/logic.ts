export interface SetupState {
  installed: boolean
  smapiInstalled: boolean
  profileCount: number
}

// First run is for a game that is not found, or a first launch with no SMAPI and no profile.
// A broken SMAPI later is the loader banner's job, and no profiles is the detail pane's empty state.
export const shouldShowFirstRun = (s: SetupState): boolean =>
  !s.installed || (!s.smapiInstalled && s.profileCount === 0)

// The Windows launch-options line that makes Steam start SMAPI instead of the game.
export const launchLine = (gameDir: string): string =>
  `"${gameDir}\\StardewModdingAPI.exe" %command%`

export const launchOptionsSet = (options: string): boolean =>
  options.toLowerCase().includes('stardewmoddingapi.exe') && options.includes('%command%')
