import type { Settings as BackendGameSettings } from '../../bindings/github.com/Rethunk-AI/mortar/internal/gamesettings/models.ts'
import type { GameSettingsValues } from './GameSettings.tsx'

export function formSettingsFromBackend(value: BackendGameSettings): GameSettingsValues | null {
  const next: GameSettingsValues = {}
  if (
    value.windowMode === 'windowed' ||
    value.windowMode === 'fullscreen' ||
    value.windowMode === 'borderless'
  ) {
    next.windowMode = value.windowMode
  }
  if (value.displayIndex !== undefined && value.displayIndex !== null) {
    next.displayIndex = value.displayIndex
  }
  if (value.preferredResolutionX !== undefined && value.preferredResolutionX !== null) {
    next.preferredResolutionX = value.preferredResolutionX
  }
  if (value.preferredResolutionY !== undefined && value.preferredResolutionY !== null) {
    next.preferredResolutionY = value.preferredResolutionY
  }
  if (value.fullscreenResolutionX !== undefined && value.fullscreenResolutionX !== null) {
    next.fullscreenResolutionX = value.fullscreenResolutionX
  }
  if (value.fullscreenResolutionY !== undefined && value.fullscreenResolutionY !== null) {
    next.fullscreenResolutionY = value.fullscreenResolutionY
  }
  if (value.zoomLevel !== undefined && value.zoomLevel !== null) {
    next.zoomLevel = value.zoomLevel
  }
  if (value.uiScale !== undefined && value.uiScale !== null) {
    next.uiScale = value.uiScale
  }
  if (value.startMuted !== undefined && value.startMuted !== null) {
    next.startMuted = value.startMuted
  }
  if (value.musicVolumeLevel !== undefined && value.musicVolumeLevel !== null) {
    next.musicVolumeLevel = value.musicVolumeLevel
  }
  if (value.soundVolumeLevel !== undefined && value.soundVolumeLevel !== null) {
    next.soundVolumeLevel = value.soundVolumeLevel
  }
  return Object.keys(next).length === 0 ? null : next
}
