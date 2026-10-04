import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/bundles/models.ts'
import type { Want } from '../queue/actions.ts'

// Downloads for bundle mods the store no longer holds; a mod with no Nexus or GitHub source cannot be fetched.
export function bundleWants(mods: Mod[] | null | undefined): Want[] {
  const wants: Want[] = []
  for (const { name, source } of mods ?? []) {
    if (source.kind === 'nexus' && source.modId) {
      wants.push({
        kind: 'install',
        name,
        modId: source.modId,
        fileId: source.fileId ?? 0,
        version: source.version ?? '',
      })
    } else if (source.kind === 'github' && source.repo) {
      wants.push({
        kind: 'install',
        name,
        repo: source.repo,
        tag: source.tag ?? '',
        asset: source.asset ?? '',
        version: source.version ?? '',
      })
    }
  }
  return wants
}
