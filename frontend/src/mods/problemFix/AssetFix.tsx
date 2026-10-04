import { useLingui } from '@lingui/react/macro'
import { Button, Tooltip } from '@mui/material'
import type { ReactNode } from 'react'
import { reportUnexpected } from '../../toasts/report.ts'
import { type Problem, sameId } from '../lookup.ts'
import { assetFixButtonStyle } from '../problemGroups.ts'
import { useMods } from '../store.ts'
import { useLocked } from '../useLocked.ts'

// Order and weight: the suggested fix first and filled, then the "make a pack win" choice, then Switch off,
// then Dismiss as the quietest action.
export function AssetFix({
  problem,
  dismissedToken,
  win,
}: {
  problem: Extract<Problem, { kind: 'asset' }>
  dismissedToken?: string | undefined
  win: (primary: boolean) => ReactNode
}) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const setEnabled = useMods((s) => s.setEnabled)
  const dismissAsset = useMods((s) => s.dismissAsset)
  const restoreDismissed = useMods((s) => s.restoreDismissed)
  const setConfigValue = useMods((s) => s.setConfigValue)
  const locked = useLocked()
  const key = problem.asset.keys?.[0]
  const id = problem.asset.packIds?.[0]
  const mod = mods.find((m) => m.key === key && (id === undefined || sameId(m.uniqueId, id)))
  const { variant, color } = assetFixButtonStyle(problem.asset.cosmetic)
  const assetButton = (label: string, onClick: () => void) => (
    <Button
      size="small"
      variant={variant}
      color={color}
      disabled={locked}
      onClick={onClick}
      sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
    >
      {label}
    </Button>
  )
  const fixes = (problem.asset.fixes ?? []).map((fix) => (
    <Tooltip
      key={`${fix.uniqueId}/${fix.field}`}
      title={t`In ${fix.name}; turns off its edits here`}
    >
      <span>
        {assetButton(t`Set ${fix.field} to ${fix.value}`, () =>
          setConfigValue(fix, fix.value).catch(reportUnexpected),
        )}
      </span>
    </Tooltip>
  ))
  const off = mod ? (
    <Button
      size="small"
      variant="outlined"
      color="inherit"
      disabled={locked}
      onClick={() => setEnabled(mod, false).catch(reportUnexpected)}
      sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
    >
      {t`Switch off`}
    </Button>
  ) : null
  const winButton = win(fixes.length === 0 && !problem.asset.cosmetic)
  if (dismissedToken !== undefined) {
    return (
      <>
        {fixes}
        {winButton}
        {off}
        <Button
          size="small"
          color="inherit"
          variant="text"
          onClick={() => restoreDismissed(dismissedToken).catch(reportUnexpected)}
          sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
        >
          {t`Restore`}
        </Button>
      </>
    )
  }
  return (
    <>
      {fixes}
      {winButton}
      {off}
      {problem.asset.kind === '' ? null : (
        <Button
          size="small"
          color="inherit"
          variant="text"
          onClick={() => {
            for (const asset of [problem.asset, ...(problem.siblings ?? [])]) {
              dismissAsset(asset).catch(reportUnexpected)
            }
          }}
          sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
        >
          {t`Dismiss`}
        </Button>
      )}
    </>
  )
}
