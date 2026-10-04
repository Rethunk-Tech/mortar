import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { reportUnexpected } from '../../toasts/report.ts'
import { LockedReason } from '../LockedReason.tsx'
import type { Problem } from '../lookup.ts'
import { useMods } from '../store.ts'
import { useLocked } from '../useLocked.ts'
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
            <LockedReason key={file.key} locked={locked}>
              <Button
                size="small"
                variant="outlined"
                color="warning"
                disabled={locked}
                onClick={() => remove(mod).catch(reportUnexpected)}
                sx={{ flexShrink: 0 }}
              >
                {t`Remove ${file.fileName || file.key}`}
              </Button>
            </LockedReason>
          ) : null
        })}
      </>
    )
  }
  return button(t`Resolve`, () => resolve(problem.duplicate))
}
