import { useLingui } from '@lingui/react/macro'
import { openSettings } from '../../nav/store.ts'
import type { Problem } from '../lookup.ts'
import type { WarningButton } from './warningButton.tsx'

// A file the game reads in the profile gets a shorter path when Mortar's data folder moves (Settings › Storage); one
// placed in the game's own folder needs the game moved, which Mortar does not do.
export function LongPathFix({
  problem,
  button,
}: {
  problem: Extract<Problem, { kind: 'longPath' }>
  button: WarningButton
}) {
  const { t } = useLingui()
  if (problem.longPath.placed) {
    return null
  }
  return button(t`Open storage settings`, () => openSettings('storage'))
}
