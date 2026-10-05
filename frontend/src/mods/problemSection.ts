import { create } from 'zustand'

interface SectionTab {
  id: string
  label: string
  count: number
  // Whether the section holds something broken, as against advice such as cleanup or harmless overlaps.
  errors: boolean
}

/** The section shown first: the first with errors, else the first with anything. */
function defaultSection(tabs: readonly SectionTab[]): string {
  return (tabs.find((t) => t.errors) ?? tabs[0])?.id ?? ''
}

/** A remembered choice still counts only while its section exists. */
function chosenSection(tabs: readonly SectionTab[], remembered: string | undefined): string {
  return tabs.some((t) => t.id === remembered) ? (remembered ?? '') : defaultSection(tabs)
}

/** The neighbouring section's id for an arrow key, stopping at the ends. */
function stepSection(tabs: readonly SectionTab[], from: string, delta: number): string {
  const at = tabs.findIndex((t) => t.id === from)
  return tabs[Math.min(Math.max(at + delta, 0), tabs.length - 1)]?.id ?? from
}

// The section chosen in each profile, kept for the session.
const useProblemSection = create<{
  byProfile: Record<string, string | undefined>
  choose: (profileId: string, id: string) => void
}>((set) => ({
  byProfile: {},
  choose: (profileId, id) => set((s) => ({ byProfile: { ...s.byProfile, [profileId]: id } })),
}))

export type { SectionTab }
export { chosenSection, defaultSection, stepSection, useProblemSection }
