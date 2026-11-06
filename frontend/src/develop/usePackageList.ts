import { useProfiles } from '../profiles/store.ts'
import { useProjectInfo } from './builds.ts'
import { useKnownPackages, usePackageDetails } from './dependencies.ts'

/** What a list of mod packages in the open profile needs to draw its rows: the packages Mortar knows, the projects it
 * builds, and the details of `ids` read in. */
export function usePackageList(game: string, ids: string[]) {
  const profile = useProfiles((s) => s.openId)
  const known = useKnownPackages((s) => s)
  const projects = useProjectInfo((s) => s.byDir)
  usePackageDetails(game, profile, ids)
  return { profile, known, projects }
}
