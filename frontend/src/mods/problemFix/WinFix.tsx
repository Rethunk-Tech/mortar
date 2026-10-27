import { useLingui } from '@lingui/react/macro'
import { Trophy, Undo2 } from 'lucide-react'
import { reportUnexpected } from '../../toasts/report.ts'
import type { Problem } from '../lookup.ts'
import { sameId } from '../lookup.ts'
import { useMods } from '../store.ts'
import { useLocked } from '../useLocked.ts'
import { applyWins } from './applyWins.ts'
import { FixAction } from './fixActions.tsx'
import { winnable } from './winnable.ts'

function WinFix({
  problem,
  primary,
}: {
  problem: Extract<Problem, { kind: 'asset' }>
  primary: boolean
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const mods = useMods((s) => s.mods)
  const { asset } = problem
  if (asset.kind !== 'edit') {
    return null
  }
  const names = asset.names ?? []
  const keys = asset.keys ?? []
  const packIds = asset.packIds ?? []
  const perEntry = asset.winnerKind === 'per-entry'
  const resolved = Boolean(asset.cosmetic && (asset.winnerKind === 'chosen' || perEntry))
  const tone = asset.cosmetic ? 'plain' : 'warning'
  if (resolved) {
    const undo = async () => {
      const winnerId = asset.winnerId
      for (const [index, id] of packIds.entries()) {
        const winnerKey = keys[index]
        if (winnerKey && (perEntry || sameId(id, winnerId))) {
          await applyWins(winnerKey, id, packIds, false)
        }
      }
    }
    return (
      <FixAction
        icon={<Undo2 size={16} aria-hidden={true} />}
        label={t`Undo`}
        weight="other"
        tone={tone}
        locked={locked}
        onClick={() => undo().catch(reportUnexpected)}
      />
    )
  }
  const choices = winnable(mods, packIds)
  if (choices.length === 0) {
    return null
  }
  return (
    <FixAction
      icon={<Trophy size={16} aria-hidden={true} />}
      label={t`Make a pack win`}
      weight={primary ? 'main' : 'other'}
      tone={tone}
      locked={locked}
      choices={choices.map((index) => ({
        label: t`Make ${{ name: names[index] ?? '' }} win`,
        onClick: () => {
          const winnerKey = keys[index]
          const winnerId = packIds[index]
          if (winnerKey && winnerId) {
            applyWins(winnerKey, winnerId, packIds, true).catch(reportUnexpected)
          }
        },
      }))}
    />
  )
}

export { WinFix }
