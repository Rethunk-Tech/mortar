import { create } from 'zustand'

/** Sections listed as links under the problems, one opened at a time: advice and history rather than problems. */
const FOOTER_SECTIONS: ReadonlySet<string> = new Set([
  'slow',
  'redundant',
  'cleanup',
  'replaced',
  'dismissed',
])

// The footer section opened in each profile, kept for the session; choosing the open one closes it.
const useProblemFooter = create<{
  byProfile: Record<string, string | undefined>
  toggle: (profileId: string, id: string) => void
}>((set) => ({
  byProfile: {},
  toggle: (profileId, id) =>
    set((s) => ({
      byProfile: { ...s.byProfile, [profileId]: s.byProfile[profileId] === id ? undefined : id },
    })),
}))

export { FOOTER_SECTIONS, useProblemFooter }
