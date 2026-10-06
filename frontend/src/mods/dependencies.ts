import type { Requirement } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'

/** One thing a mod needs: a page to open when it is a mod on the same site, otherwise just its name. */
interface Dependency {
  name: string
  url?: string
  note?: string
}

/** A Nexus page's requirements: its own mods open their page, anything outside Nexus is named only. */
function nexusDependencies(reqs: Requirement[] | null | undefined): Dependency[] {
  return (reqs ?? []).map((r) => ({
    name: r.name,
    note: r.notes,
    ...(r.external ? {} : { url: r.url }),
  }))
}

export { type Dependency, nexusDependencies }
