import type { ProjectInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'

// A report row is the project's when it carries the mod's name or its plugin GUID: the bridge names a plugin by its
// package once Mortar knows the package, by its GUID before.
export function isProjectRow(info: ProjectInfo, name: string, id = ''): boolean {
  const guid = info.guid.toLowerCase()
  const lower = name.toLowerCase()
  return (
    lower === info.name.toLowerCase() ||
    lower === guid ||
    (guid !== '' && id.toLowerCase().endsWith(guid))
  )
}
