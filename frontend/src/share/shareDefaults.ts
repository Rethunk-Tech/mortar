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
