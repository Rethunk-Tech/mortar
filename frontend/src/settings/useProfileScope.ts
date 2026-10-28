import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { useSettingsScope } from './settingsScope.ts'

/** Whether the page edits the open profile's own values: the profile scope is on and a profile is open to hold them. */
export function useProfileScope() {
  const scoped = useSettingsScope((s) => s.profile)
  const open = useProfiles((s) => openProfileOf(s) !== undefined)
  return scoped && open
}
