import { createContext, useContext } from 'react'

export const SettingsSearchContext = createContext('')

export function useSettingsSearch(): string {
  return useContext(SettingsSearchContext)
}
