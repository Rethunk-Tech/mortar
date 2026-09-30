import { Browser } from '@wailsio/runtime'
import { BugURL } from '../../bindings/github.com/Rethunk-AI/mortar/internal/support/service.ts'
import { reportUnexpected } from '../toasts/report.ts'

// Opens a new Mortar issue prefilled with the version, OS and game; game is '' outside a game's pages.
export function reportBug(game: string): void {
  BugURL(game)
    .then((url) => Browser.OpenURL(url))
    .catch(reportUnexpected)
}
