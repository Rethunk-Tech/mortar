import { useLingui } from '@lingui/react/macro'
import { MakePackageWin } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { openProfileOf, useProfiles } from '../../profiles/store.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import type { Problem } from '../lookup.ts'
import { useMods } from '../store.ts'
import type { WarningButton } from './warningButton.tsx'

// Make … win moves a beaten package to just after the winner in package order, so it takes those files instead.
export function FileConflictFix({
  problem,
  button,
}: {
  problem: Extract<Problem, { kind: 'fileConflict' }>
  button: WarningButton
}) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const { winner, beaten } = problem.fileConflict
  const makeWin = (loser: string) => {
    const s = useProfiles.getState()
    const open = openProfileOf(s)
    if (s.game && open) {
      MakePackageWin(s.game.id, open.id, loser, winner)
        .then((p) => s.replace(p))
        .catch(reportUnexpected)
    }
  }
  return beaten.map((loser) => {
    const name = mods.find((m) => m.key === loser)?.name ?? loser
    return <span key={loser}>{button(t`Make ${{ name }} win`, () => makeWin(loser), true)}</span>
  })
}
