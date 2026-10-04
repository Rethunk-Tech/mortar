import { SetGameSettings } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SetOverrides } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { applyStagedCover, type StagedCover } from '../game/cover.ts'
import { errorMessage, reportError, reportUnexpected } from '../toasts/report.ts'
import { clipDescription } from './appearance.ts'
import type { GameSettingsValues } from './GameSettings.tsx'
import { useProfiles } from './store.ts'

type LaunchError = { field: 'options' | 'settings'; message: string } | null

async function coverSaved(
  gameId: string,
  profile: Profile,
  staged: StagedCover,
  failure: string,
): Promise<boolean> {
  try {
    const next = await applyStagedCover(gameId, profile.id, staged)
    if (next) {
      useProfiles.getState().replace(next)
    }
    return true
  } catch (error) {
    reportError(failure)(error)
    return false
  }
}

export async function saveProfile({
  profile,
  gameId,
  launchOptions,
  launchPrefix,
  launchEnv,
  overrides,
  stagedCover,
  gameSettings,
  color,
  icon,
  description,
  setLaunchOptions,
  setLaunchSettings,
  setAppearance,
  setLaunchError,
  setBusy,
  onClose,
  coverFailure,
}: {
  profile: Profile
  gameId: string
  launchOptions: string
  launchPrefix: string
  launchEnv: string
  overrides: Record<string, string>
  stagedCover: StagedCover
  gameSettings: GameSettingsValues | null
  color: string
  icon: string
  description: string
  setLaunchOptions: (id: string, options: string) => Promise<void>
  setLaunchSettings: (id: string, prefix: string, env: string) => Promise<void>
  setAppearance: (id: string, color: string, icon: string, description: string) => Promise<void>
  setLaunchError: (value: LaunchError) => void
  setBusy: (value: boolean) => void
  onClose: () => void
  coverFailure: string
}) {
  setBusy(true)
  setLaunchError(null)
  try {
    try {
      await setLaunchOptions(profile.id, launchOptions)
    } catch (error) {
      setLaunchError({ field: 'options', message: errorMessage(error) })
      return
    }
    try {
      await setLaunchSettings(profile.id, launchPrefix, launchEnv)
    } catch (error) {
      setLaunchError({ field: 'settings', message: errorMessage(error) })
      return
    }
    try {
      await setAppearance(profile.id, color, icon, clipDescription(description))
      if (gameId) {
        await SetGameSettings(gameId, profile.id, gameSettings ?? {})
        useProfiles.getState().replace(await SetOverrides(gameId, profile.id, overrides))
      }
    } catch (error) {
      reportUnexpected(error)
      return
    }
    if (await coverSaved(gameId, profile, stagedCover, coverFailure)) {
      onClose()
    }
  } finally {
    setBusy(false)
  }
}
