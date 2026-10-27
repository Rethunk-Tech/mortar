import { create } from 'zustand'
import { Search } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import { AddDependency } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/source/models.ts'
import { download } from '../queue/actions.ts'
import { useProjectInfo } from './builds.ts'

export const THUNDERSTORE = 'thunderstore'
export const NO_FILTER = {
  include: [],
  exclude: [],
  sort: 'downloads',
  installed: '',
  obsolete: '',
  broken: '',
}

export type Known = Record<string, Item>

// The packages looked up so far, held outside the wizard: one arriving redraws the step that lists it, not the wizard.
export const useKnownPackages = create<Known>(() => ({}))
export const knowPackage = (item: Item) => useKnownPackages.setState({ [item.id]: item })

export async function findPackage(
  game: string,
  profile: string,
  pkg: string,
): Promise<Item | undefined> {
  const name = pkg.slice(pkg.indexOf('-') + 1).replaceAll('_', ' ')
  const page = await Search(game, THUNDERSTORE, name, 1, profile, NO_FILTER)
  return (page.items ?? []).find((i) => i.id === pkg)
}

/** Records each package in the new project and installs it into the open (development) profile. */
export async function startDependencies(
  game: string,
  dir: string,
  { ids, optional }: { ids: string[]; optional: string[] },
  known: Known,
) {
  if (ids.length === 0) {
    return
  }
  for (const id of ids) {
    const version = known[id]?.version || (await findPackage(game, '', id))?.version
    if (version) {
      await AddDependency(game, dir, id, version, optional.includes(id))
    }
  }
  // A package another project here publishes is built from that project, so it is not downloaded.
  const local = new Set(Object.values(useProjectInfo.getState().byDir).map((info) => info.package))
  const fetched = ids.filter((id) => !local.has(id))
  if (fetched.length > 0) {
    await download(fetched.map((id) => ({ kind: 'install', package: id })))
  }
}
