interface LaunchPresetFields {
  options: string
  prefix: string
  env: string
}

function applyLaunchPreset(preset: Partial<LaunchPresetFields>): LaunchPresetFields {
  return {
    options: preset.options ?? '',
    prefix: preset.prefix ?? '',
    env: preset.env ?? '',
  }
}

export { applyLaunchPreset }
