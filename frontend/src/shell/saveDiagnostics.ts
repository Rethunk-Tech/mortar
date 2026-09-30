import { SaveDiagnostics } from '../../bindings/github.com/Rethunk-AI/mortar/internal/support/service.ts'
import { reportUnexpected } from '../toasts/report.ts'

// Asks where to write a redacted diagnostics zip; game and profile may be ''.
export function saveDiagnostics(game: string, profile: string): void {
  SaveDiagnostics(game, profile).catch(reportUnexpected)
}
