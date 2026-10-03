import type { ReactNode } from 'react'
import { SettingsSearchContext } from './useSettingsSearch.ts'

export function SettingsSearchProvider({
  query,
  children,
}: {
  query: string
  children: ReactNode
}) {
  return <SettingsSearchContext.Provider value={query}>{children}</SettingsSearchContext.Provider>
}
