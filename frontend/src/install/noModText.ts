import { useLingui } from '@lingui/react/macro'
import { useProfileLoader } from '../profiles/store.ts'

// useNoModText asks for the mod folder of an archive whose mod sits somewhere Mortar does not look.
export function useNoModText(profileId?: string): string {
  const { t } = useLingui()
  const loader = useProfileLoader(profileId)?.name || t`the mod loader`
  return t`This archive has no ${loader} mod where Mortar expects one. Pick the folder that holds the mod.`
}
