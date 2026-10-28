import { useEffect } from 'react'
import { create } from 'zustand'
import { Search } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import type { ProjectInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
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

/** Records each package in the new project and installs it into the open (development) profile; returns the ids it
 * skipped for having no known version. */
export async function startDependencies(
  game: string,
  dir: string,
  { ids, optional }: { ids: string[]; optional: string[] },
  known: Known,
): Promise<string[]> {
  const skipped: string[] = []
  if (ids.length === 0) {
    return skipped
  }
  const projects = Object.values(useProjectInfo.getState().byDir)
  for (const id of ids) {
    // A package one of the player's projects publishes may not be listed yet; that project's version stands in.
    const version =
      known[id]?.version ||
      (await findPackage(game, '', id))?.version ||
      projects.find((info) => info.package === id)?.version
    if (version) {
      await AddDependency(game, dir, id, version, optional.includes(id))
    } else {
      skipped.push(id)
    }
  }
  // A package another project here publishes is built from that project, so it is not downloaded.
  const local = new Set(projects.map((info) => info.package))
  const fetched = ids.filter((id) => !(local.has(id) || skipped.includes(id)))
  if (fetched.length > 0) {
    await download(fetched.map((id) => ({ kind: 'install', package: id })))
  }
  return skipped
}

export type Shown = Pick<Item, 'name' | 'author'> & { picture?: string }

// A package that is one of the player's own projects shows as that project until the source lists it.
export function ownProject(byDir: Record<string, ProjectInfo>, id: string): Shown | undefined {
  const info = Object.values(byDir).find((i) => i.package === id)
  return (
    info && { name: info.name, author: info.author, ...(info.icon ? { picture: info.icon } : {}) }
  )
}

// Looks up each listed package not known yet, so its row can show its picture and author.
export function usePackageDetails(game: string, profile: string, ids: string[]) {
  const missing = useKnownPackages((known) => ids.filter((id) => !known[id]).join(','))
  useEffect(() => {
    for (const id of missing.split(',').filter(Boolean)) {
      findPackage(game, profile, id)
        .then((item) => item && knowPackage(item))
        .catch(() => undefined)
    }
  }, [game, profile, missing])
}
