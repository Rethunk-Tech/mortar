import { useLingui } from '@lingui/react/macro'
import { Button, Menu, MenuItem } from '@mui/material'
import { type MouseEvent, useState } from 'react'
import { DisabledReason } from '../../shell/DisabledReason.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import type { Problem } from '../lookup.ts'
import { sameId } from '../lookup.ts'
import { useMods } from '../store.ts'
import { useLocked } from '../useLocked.ts'
import { applyWins } from './applyWins.ts'
import { winnable } from './winnable.ts'

const buttonHeight = 28

function WinFix({
  problem,
  primary,
}: {
  problem: Extract<Problem, { kind: 'asset' }>
  primary: boolean
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const mods = useMods((s) => s.mods)
  const { asset } = problem
  if (asset.kind !== 'edit') {
    return null
  }
  const names = asset.names ?? []
  const keys = asset.keys ?? []
  const packIds = asset.packIds ?? []
  const winnerLabel = asset.winnerName ?? ''
  const perEntry = winnerLabel === 'decided per entry'
  const resolved = Boolean(asset.cosmetic && (winnerLabel.endsWith(' wins') || perEntry))
  const lockedTitle = t`Stop the game to change mods.`
  const sx = { height: buttonHeight, whiteSpace: 'nowrap', flexShrink: 0 } as const
  const undo = async () => {
    const winnerId = asset.winnerId ?? ''
    for (const [index, id] of packIds.entries()) {
      const winnerKey = keys[index]
      if (winnerKey && (perEntry || sameId(id, winnerId))) {
        await applyWins(winnerKey, id, packIds, false)
      }
    }
  }
  const choose = (index: number) => {
    setAnchor(null)
    const winnerKey = keys[index]
    const winnerId = packIds[index]
    if (winnerKey && winnerId) {
      applyWins(winnerKey, winnerId, packIds, true).catch(reportUnexpected)
    }
  }
  const choices = resolved ? [] : winnable(mods, packIds)
  if (!resolved && choices.length === 0) {
    return null
  }
  const control = resolved ? (
    <Button
      size="small"
      variant="outlined"
      disabled={locked}
      onClick={() => undo().catch(reportUnexpected)}
      sx={sx}
    >
      {t`Undo`}
    </Button>
  ) : (
    <>
      <Button
        size="small"
        variant={primary ? 'contained' : 'outlined'}
        color={primary ? 'warning' : 'inherit'}
        disabled={locked}
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        onClick={(e: MouseEvent<HTMLButtonElement>) => setAnchor(e.currentTarget)}
        sx={sx}
      >
        {t`Make a pack win`}
      </Button>
      <Menu anchorEl={anchor} open={Boolean(anchor)} onClose={() => setAnchor(null)}>
        {choices.map((index) => {
          const name = names[index] ?? ''
          return (
            <MenuItem
              key={`${keys[index] ?? index}/${packIds[index] ?? name}`}
              onClick={() => choose(index)}
            >
              {t`Make ${name} win`}
            </MenuItem>
          )
        })}
      </Menu>
    </>
  )
  return (
    <DisabledReason title={lockedTitle} disabled={locked}>
      {control}
    </DisabledReason>
  )
}

export { WinFix }
