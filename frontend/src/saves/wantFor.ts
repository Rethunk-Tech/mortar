import type { Lack } from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/models.ts'
import type { Want } from '../queue/actions.ts'

export function wantFor(lack: Lack): Want | null {
  const { where } = lack
  if (where?.site === 'GitHub' && where.github) {
    return { kind: 'install', repo: where.github, name: where.github }
  }
  if (where?.site === 'Nexus' && where.pageId > 0) {
    return {
      kind: 'install',
      modId: where.pageId,
      latest: true,
      fileId: where.fileId,
      name: lack.name,
      fileName: where.fileName,
      version: where.version,
    }
  }
  return null
}
