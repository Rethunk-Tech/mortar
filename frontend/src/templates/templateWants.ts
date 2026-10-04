import type { Template } from '../../bindings/github.com/Rethunk-AI/mortar/internal/templates/models.ts'
import type { Want } from '../queue/actions.ts'

type Mod = NonNullable<Template['bundle']>[number]

function wantFor(mod: Mod): Want | null {
  const { source } = mod
  if (source.kind === 'github' && source.repo) {
    return {
      kind: 'install',
      repo: source.repo,
      tag: source.tag ?? '',
      asset: source.asset ?? '',
      name: source.repo,
    }
  }
  if (source.kind === 'nexus' && (source.modId ?? 0) > 0) {
    return {
      kind: 'install',
      modId: source.modId ?? 0,
      fileId: source.fileId ?? 0,
      version: source.version ?? '',
      name: mod.name,
    }
  }
  return null
}

/** The downloads for the mods a template could not add from the store, one per archive the template recorded. */
export function templateWants(
  template: Pick<Template, 'bundle'>,
  missing: readonly string[],
): Want[] {
  const wants = new Map<string, Want>()
  for (const mod of template.bundle ?? []) {
    const want = missing.includes(mod.name) && !wants.has(mod.entryKey) ? wantFor(mod) : null
    if (want) {
      wants.set(mod.entryKey, want)
    }
  }
  return [...wants.values()]
}
