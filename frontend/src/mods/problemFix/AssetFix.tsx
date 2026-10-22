import { useLingui } from '@lingui/react/macro'
import type { ReactNode } from 'react'
import { reportUnexpected } from '../../toasts/report.ts'
import { type Problem, sameId } from '../lookup.ts'
import { useMods } from '../store.ts'
import { useLocked } from '../useLocked.ts'
import { FixAction } from './fixActions.tsx'
import { RestoreButton } from './RestoreButton.tsx'

// Order and weight: the suggested setting first, then the "make a pack win" choice, then Disable, then Dismiss.
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
  const setConfigValue = useMods((s) => s.setConfigValue)
  const locked = useLocked()
  const key = problem.asset.keys?.[0]
  const id = problem.asset.packIds?.[0]
  const mod = mods.find((m) => m.key === key && (id === undefined || sameId(m.id, id)))
  const tone = problem.asset.cosmetic ? 'plain' : 'warning'
  const fixes = problem.asset.fixes ?? []
  return (
    <>
      {fixes.map((fix) => (
        <FixAction
          key={`${fix.id}/${fix.field}`}
          label={t`Set ${{ field: fix.field }} to ${{ value: fix.value }}`}
          tooltip={t`In ${{ name: fix.name }}; disables its edits here`}
          weight="main"
          tone={tone}
          locked={locked}
          onClick={() => setConfigValue(fix, fix.value).catch(reportUnexpected)}
        />
      ))}
      {win(fixes.length === 0 && !problem.asset.cosmetic)}
      {mod ? (
        <FixAction
          label={t`Disable`}
          weight="other"
          tone={tone}
          locked={locked}
          onClick={() => setEnabled(mod, false).catch(reportUnexpected)}
        />
      ) : null}
      {dismissedToken === undefined ? null : <RestoreButton token={dismissedToken} />}
      {dismissedToken === undefined && problem.asset.kind !== '' ? (
        <FixAction
          label={t`Dismiss`}
          weight="quiet"
          onClick={() => {
            for (const asset of [problem.asset, ...(problem.siblings ?? [])]) {
              dismissAsset(asset).catch(reportUnexpected)
            }
          }}
        />
      ) : null}
    </>
  )
}
