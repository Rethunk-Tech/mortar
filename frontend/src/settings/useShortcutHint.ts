import { mergeBindings, type ShortcutId } from './shortcuts.ts'
import { useSettings } from './store.ts'

// The chord an action is bound to now, for a menu row's trailing hint; empty when the user unbound it.
export function useShortcutHint(id: ShortcutId): string {
  return mergeBindings(useSettings((s) => s.shortcuts))[id]
}
