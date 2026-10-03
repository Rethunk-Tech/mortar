import { useLingui } from '@lingui/react/macro'
import { Button, Menu, MenuItem } from '@mui/material'
import { type MouseEvent, useState } from 'react'
import type { Profile } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { SetWinner } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useProfiles } from '../../profiles/store.ts'
import { DisabledReason } from '../../shell/DisabledReason.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import type { Problem } from '../lookup.ts'
import { useMods } from '../store.ts'
import { useLocked } from '../useLocked.ts'

const buttonHeight = 28

function applyWins(winnerKey: string, packIds: string[], skip: number, on: boolean) {
  const { game, openId } = useProfiles.getState()
  const { loadProblems } = useMods.getState()
  const { replace } = useProfiles.getState()
  if (!(game && openId)) {
    return
  }
  const run = async () => {
    let profile: Profile | undefined
    for (const [i, id] of packIds.entries()) {
      if (i !== skip && id) {
        profile = await SetWinner(game.id, openId, winnerKey, id, on)
      }
    }
    if (!profile) {
      return
    }
    replace(profile)
    await loadProblems()
  }
  run().catch(reportUnexpected)
}

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
  const { asset } = problem
  if (asset.kind !== 'edit') {
    return null
  }
  const names = asset.names ?? []
  const keys = asset.keys ?? []
  const packIds = asset.packIds ?? []
  const winnerLabel = asset.winnerName ?? ''
  const resolved = Boolean(asset.cosmetic && winnerLabel.endsWith(' wins'))
  const lockedTitle = t`Stop the game to change mods.`
  const sx = { height: buttonHeight, whiteSpace: 'nowrap', flexShrink: 0 } as const
  const undo = () => {
    const winnerId = asset.winnerId ?? ''
    const index = packIds.findIndex((id) => id.toLowerCase() === winnerId.toLowerCase())
    const winnerKey = keys[index]
    if (winnerKey) {
      applyWins(winnerKey, packIds, index, false)
    }
  }
  const choose = (index: number) => {
    setAnchor(null)
    const winnerKey = keys[index]
    if (winnerKey) {
      applyWins(winnerKey, packIds, index, true)
    }
  }
  const control = resolved ? (
    <Button size="small" variant="outlined" disabled={locked} onClick={undo} sx={sx}>
      {t`Undo`}
    </Button>
  ) : (
    <>
      <Button
        size="small"
        variant={primary ? 'contained' : 'outlined'}
        color={primary ? 'warning' : 'inherit'}
        disabled={locked}
        onClick={(e: MouseEvent<HTMLButtonElement>) => setAnchor(e.currentTarget)}
        sx={sx}
      >
        {t`Make a pack win`}
      </Button>
      <Menu anchorEl={anchor} open={Boolean(anchor)} onClose={() => setAnchor(null)}>
        {names.map((name, index) => (
          <MenuItem
            key={`${keys[index] ?? index}/${packIds[index] ?? name}`}
            onClick={() => choose(index)}
          >
            {t`Make ${name} win`}
          </MenuItem>
        ))}
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
