import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { currentGame } from '../nav/currentGame.ts'
import { useSettings } from '../settings/store.ts'

/** Whether a profile entry is a mod built by one of the game's Develop projects: installed from that project's build
 * folder, so its details belong to the project's files and its editing to the Develop tab. */
export function isDevelopEntry(entry: Entry | undefined): boolean {
  const folder = entry?.source.kind === 'local' ? (entry.source.folder ?? '') : ''
  if (folder === '') {
    return false
  }
  const projects = useSettings.getState().games?.[currentGame()]?.devProjects ?? []
  return projects.some(
    (p) =>
      folder === p.output ||
      folder.startsWith(`${p.output}/`) ||
      folder.startsWith(`${p.output}\\`),
  )
}
