import { plural } from '@lingui/core/macro'
import type { GitState } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'

// What the repository has not committed or pushed; nothing when both are clean.
export function gitSummary(git: GitState | null | undefined): string {
  if (!git) {
    return ''
  }
  return [
    git.changed > 0 ? plural({ count: git.changed }, { one: '# changed', other: '# changed' }) : '',
    git.unpushed > 0
      ? plural({ count: git.unpushed }, { one: '# to push', other: '# to push' })
      : '',
  ]
    .filter(Boolean)
    .join(', ')
}
