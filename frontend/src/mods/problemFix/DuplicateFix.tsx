import { useLingui } from '@lingui/react/macro'
import { Trash2 } from 'lucide-react'
import { reportUnexpected } from '../../toasts/report.ts'
import type { Problem } from '../lookup.ts'
import { useMods } from '../store.ts'
import { useLocked } from '../useLocked.ts'
import { FixAction } from './fixActions.tsx'
import type { WarningButton } from './warningButton.tsx'

export function DuplicateFix({
  problem,
  button,
}: {
  problem: Extract<Problem, { kind: 'duplicate' }>
  button: WarningButton
}) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const resolve = useMods((s) => s.resolve)
  const remove = useMods((s) => s.remove)
  const locked = useLocked()
  const nexusFiles = (problem.duplicate.nexusFiles ?? []).filter((file) => file.remove)
  if (nexusFiles.length > 0) {
    return (
      <>
        {nexusFiles.map((file) => {
          const mod = mods.find((candidate) => candidate.key === file.key)
          return mod ? (
            <FixAction
              icon={<Trash2 size={16} aria-hidden={true} />}
              key={file.key}
              label={t`Remove ${{ name: file.fileName || file.key }}`}
              weight="other"
              locked={locked}
              onClick={() => remove(mod).catch(reportUnexpected)}
            />
          ) : null
        })}
      </>
    )
  }
  return button(t`Resolve`, () => resolve(problem.duplicate))
}
