import { useLingui } from '@lingui/react/macro'
import { Button, Tooltip } from '@mui/material'
import { reportUnexpected } from '../../toasts/report.ts'
import { type Problem, sameId } from '../lookup.ts'
import { assetFixButtonStyle } from '../problemGroups.ts'
import { useMods } from '../store.ts'
import { useLocked } from '../useLocked.ts'

export function AssetFix({
  problem,
  dismissedToken,
}: {
  problem: Extract<Problem, { kind: 'asset' }>
  dismissedToken?: string | undefined
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
  const off = mod
    ? assetButton(t`Switch off`, () => setEnabled(mod, false).catch(reportUnexpected))
    : null
  if (dismissedToken !== undefined) {
    return (
      <>
        {fixes}
        {off}
        <Button
          size="small"
          color="info"
          variant="outlined"
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
      {off}
      {problem.asset.kind === '' ? null : (
        <Button
          size="small"
          color="info"
          variant="outlined"
          onClick={() => dismissAsset(problem.asset).catch(reportUnexpected)}
          sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
        >
          {t`Dismiss`}
        </Button>
      )}
    </>
  )
}
