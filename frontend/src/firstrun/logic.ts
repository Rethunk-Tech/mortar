export interface SetupState {
  installed: boolean
  smapiReady: boolean
  profileCount: number
}

// First run shows until the game is found, SMAPI is in place and a profile exists.
export const shouldShowFirstRun = (s: SetupState): boolean =>
  !(s.installed && s.smapiReady) || s.profileCount === 0

// The Windows launch-options line that makes Steam start SMAPI instead of the game.
export const launchLine = (gameDir: string): string =>
  `"${gameDir}\\StardewModdingAPI.exe" %command%`

export const launchOptionsSet = (options: string): boolean =>
  options.toLowerCase().includes('stardewmoddingapi.exe')
