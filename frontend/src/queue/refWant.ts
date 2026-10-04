import type { Ref } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type { Want } from './actions.ts'

// The Want for a problem Ref, or null when the Ref names nothing downloadable (no GitHub repo, no Nexus page).
export function refWant(where: Ref, kind: 'dependency' | 'install', name?: string): Want | null {
  if (where.site === 'GitHub' && where.github !== '') {
    return { kind, repo: where.github, name: where.github }
  }
  if (where.site === 'Nexus' && where.pageId > 0) {
    return {
      kind,
      modId: where.pageId,
      latest: true,
      fileId: where.fileId,
      name: name ?? where.pageName,
      fileName: where.fileName,
      version: where.version,
    }
  }
  return null
}
