import type {
  LaunchPreset,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

// The backend resolves this name to the profile's own launch settings.
const BASE_PRESET = 'Standard'

interface PlayPreset {
  /** What to pass to the backend: the preset's id, or BASE_PRESET. */
  key: string
  name: string
  isDefault: boolean
  base: boolean
}

// Empty while the profile has no named presets, so the Play menu stays as it was.
function playPresets(profile: Profile | undefined): PlayPreset[] {
  const named = profile?.launchPresets ?? []
  if (!profile || named.length === 0) {
    return []
  }
  const current = profile.defaultLaunchPreset ?? ''
  const hasDefault = named.some((p) => p.id === current)
  return [
    { key: BASE_PRESET, name: BASE_PRESET, isDefault: !hasDefault, base: true },
    ...named.map((p) => ({ key: p.id, name: p.name, isDefault: p.id === current, base: false })),
  ]
}

function copyName(name: string, existing: string[]): string {
  const taken = new Set(existing.map((n) => n.toLowerCase()))
  let candidate = `${name} copy`
  for (let n = 2; taken.has(candidate.toLowerCase()); n++) {
    candidate = `${name} copy ${n}`
  }
  return candidate
}

function duplicatePreset(preset: LaunchPreset, existing: LaunchPreset[]): LaunchPreset {
  return {
    ...preset,
    id: '',
    name: copyName(
      preset.name,
      existing.map((p) => p.name),
    ),
  }
}

// A name the backend would refuse: empty, the base preset's, or one another preset has.
function presetNameError(
  name: string,
  existing: LaunchPreset[],
  editingId: string,
): 'empty' | 'taken' | null {
  const key = name.trim().toLowerCase()
  if (key === '') {
    return 'empty'
  }
  if (
    key === BASE_PRESET.toLowerCase() ||
    existing.some((p) => p.id !== editingId && p.name.trim().toLowerCase() === key)
  ) {
    return 'taken'
  }
  return null
}

export { BASE_PRESET, copyName, duplicatePreset, type PlayPreset, playPresets, presetNameError }
