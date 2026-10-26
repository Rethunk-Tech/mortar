import { useLayoutEffect } from 'react'
import { create } from 'zustand'
import { prefMatches } from './prefFilter.ts'

// The settings search, held outside React so each row selects only its own answer: typing re-renders the rows whose
// match changed, not every row on the page.
const useSettingsQuery = create<{ query: string }>(() => ({ query: '' }))

export function useSettingsSearch(): string {
  return useSettingsQuery((s) => s.query)
}

export function useSettingsSearching(): boolean {
  return useSettingsQuery((s) => s.query.trim() !== '')
}

export function useSettingsMatch(label: string, description = ''): boolean {
  return useSettingsQuery((s) => prefMatches(s.query, label, description))
}

/** Whether any of a component's row labels matches the search (always, with no search): what a component reads from
 * the backend for rows a search hides is not read, so a search mounts every section without asking for their data. */
export function useSettingsRelevant(...labels: string[]): boolean {
  return useSettingsQuery((s) => labels.some((label) => prefMatches(s.query, label)))
}

export const setSettingsQuery = (query: string) => useSettingsQuery.setState({ query })

export const settingsQuery = () => useSettingsQuery.getState().query

/** Starts the shell's search at initial and clears it when the shell closes. */
export function useShellQuery(initial: string) {
  useLayoutEffect(() => {
    setSettingsQuery(initial)
    return () => setSettingsQuery('')
  }, [initial])
}
