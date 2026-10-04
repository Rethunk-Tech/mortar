import type { Lack } from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/models.ts'
import { refWant, type Want } from '../queue/actions.ts'

export function wantFor(lack: Lack): Want | null {
  return lack.where ? refWant(lack.where, 'install', lack.name) : null
}
