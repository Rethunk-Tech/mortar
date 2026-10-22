import { useLingui } from '@lingui/react/macro'
import { MakePackageWin } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { openProfileOf, useProfiles } from '../../profiles/store.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import type { Problem } from '../lookup.ts'
import { useMods } from '../store.ts'
import { useLocked } from '../useLocked.ts'
import { FixAction } from './fixActions.tsx'
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
  const locked = useLocked()
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
  const nameOf = (loser: string) => mods.find((m) => m.key === loser)?.name ?? loser
  const [only] = beaten
  if (beaten.length === 1 && only !== undefined) {
    return button(t`Make ${{ name: nameOf(only) }} win`, () => makeWin(only))
  }
  return (
    <FixAction
      label={t`Make a package win`}
      weight="main"
      locked={locked}
      choices={beaten.map((loser) => ({
        label: t`Make ${{ name: nameOf(loser) }} win`,
        onClick: () => makeWin(loser),
      }))}
    />
  )
}
