import { useLingui } from '@lingui/react/macro'
import { Download, type LucideIcon } from 'lucide-react'
import { openImport } from '../share/store.ts'

export interface ImportEntry {
  key: string
  label: string
  icon: LucideIcon
  run: () => void
}

// The one way into Import from the profile menus: its dialog then offers every place a profile can come from.
export function useProfileImport(): ImportEntry[] {
  const { t } = useLingui()
  return [{ key: 'import', label: t`Import…`, icon: Download, run: () => openImport() }]
}
