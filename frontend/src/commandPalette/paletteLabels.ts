import type { PaletteLabels } from './items.ts'

export function paletteActionLabels(
  t: (template: TemplateStringsArray, ...args: unknown[]) => string,
): PaletteLabels {
  return {
    play: t`Play`,
    updates: t`Check for mod updates`,
    downloads: t`Open Downloads`,
    import: t`Import`,
    pasteLink: t`Paste a link…`,
    collectionReview: t`Review collection update`,
    findCrashCause: t`Find the crash cause`,
    configureMod: (name) => t`Configure ${name}…`,
    share: t`Share`,
    newProfile: t`New profile`,
    streamOverlay: t`Stream overlay`,
    recentChanges: t`Recent changes`,
    diagnostics: t`Diagnostics`,
    profileHint: t`Open profile`,
    modHint: t`Open mod`,
    settingsHint: t`Settings`,
    tabs: {
      mods: t`Go to Mods`,
      problems: t`Go to Problems`,
      'load-order': t`Go to Load order`,
      saves: t`Go to Saves`,
      notes: t`Go to Notes`,
      console: t`Go to Console`,
      performance: t`Go to Performance`,
    },
    toggle: (name) => t`Toggle ${name}`,
  }
}
