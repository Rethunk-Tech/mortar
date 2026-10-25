import { Search } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import { AddDependency } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/source/models.ts'
import { download } from '../queue/actions.ts'

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
export async function startDependencies(game: string, dir: string, ids: string[], known: Known) {
  if (ids.length === 0) {
    return
  }
  for (const id of ids) {
    const version = known[id]?.version || (await findPackage(game, '', id))?.version
    if (version) {
      await AddDependency(game, dir, id, version)
    }
  }
  await download(ids.map((id) => ({ kind: 'install', package: id })))
}
