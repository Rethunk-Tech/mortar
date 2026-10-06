export interface ShareInclude {
  disabledMods: boolean
  fomodChoices: boolean
  notes: boolean
  configFiles: boolean
}

export function shareIncludeDefaults(s: {
  shareIncludeDisabledMods?: boolean | null
  shareIncludeFomodChoices?: boolean | null
  shareIncludeNotes?: boolean | null
  shareIncludeConfigFiles?: boolean | null
}): ShareInclude {
  return {
    disabledMods: s.shareIncludeDisabledMods === true,
    fomodChoices: s.shareIncludeFomodChoices !== false,
    notes: s.shareIncludeNotes !== false,
    configFiles: s.shareIncludeConfigFiles !== false,
  }
}

export function toShareInclude(value: ShareInclude) {
  return {
    disabledMods: value.disabledMods,
    fomodChoices: value.fomodChoices,
    notes: value.notes,
    configFiles: value.configFiles,
  }
}

// FOMOD installers come from Nexus, so the option means nothing on a game modded only from elsewhere unless an
// entry already carries choices (an archive installed by hand).
export function offersFomod(
  entries: readonly { fomod?: object | null }[] | null | undefined,
  sources: readonly string[] | null | undefined,
): boolean {
  return (
    (sources ?? []).includes('nexus') ||
    (entries ?? []).some((e) => Object.keys(e.fomod ?? {}).length > 0)
  )
}
