import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/bundles/models.ts'
import type { Want } from '../queue/actions.ts'

// The download for one bundle mod; a mod with no Nexus or GitHub source cannot be fetched.
function bundleModWant({ name, source }: Mod): Want | null {
  if (source.kind === 'nexus' && source.modId) {
    return {
      kind: 'install',
      name,
      modId: source.modId,
      fileId: source.fileId ?? 0,
      version: source.version ?? '',
    }
  }
  if (source.kind === 'github' && source.repo) {
    return {
      kind: 'install',
      name,
      repo: source.repo,
      tag: source.tag ?? '',
      asset: source.asset ?? '',
      version: source.version ?? '',
    }
  }
  return null
}

// Downloads for bundle or template mods the store no longer holds, one per archive.
export function bundleWants(mods: Mod[] | null | undefined): Want[] {
  const wants = new Map<string, Want>()
  for (const mod of mods ?? []) {
    const want = wants.has(mod.entryKey) ? null : bundleModWant(mod)
    if (want) {
      wants.set(mod.entryKey, want)
    }
  }
  return [...wants.values()]
}
