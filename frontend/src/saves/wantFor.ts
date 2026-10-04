import type { Lack } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/models.ts'
import type { Want } from '../queue/actions.ts'
import { refWant } from '../queue/refWant.ts'

export function wantFor(lack: Lack): Want | null {
  return lack.where ? refWant(lack.where, 'install', lack.name) : null
}
