import { useLingui } from '@lingui/react/macro'
import { Box, Link, Typography } from '@mui/material'
import { memo, useState } from 'react'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { openSettings } from '../nav/store.ts'
import { AddGameDialog } from './AddGameDialog.tsx'

// Under the list: adding a game of your own, the hidden games, and where to look when none was found.
export const ListFooter = memo(function ListFooterView({
  games,
  hiddenCount,
  showHidden,
  onShowHidden,
  onChanged,
}: {
  games: GameInfo[]
  hiddenCount: number
  showHidden: boolean
  onShowHidden: (show: boolean) => void
  onChanged: () => void
}) {
  const { t } = useLingui()
  const [adding, setAdding] = useState(false)
  return (
    <>
      <Box sx={{ px: 2, py: 0.75 }}>
        <Link component="button" onClick={() => setAdding(true)} sx={{ fontSize: 14 }}>
          {t`Add a game…`}
        </Link>
      </Box>
      <AddGameDialog
        open={adding}
        onClose={() => {
          setAdding(false)
          onChanged()
        }}
      />
      {hiddenCount > 0 ? (
        <Box sx={{ px: 2, py: 0.75 }}>
          <Link component="button" onClick={() => onShowHidden(!showHidden)} sx={{ fontSize: 14 }}>
            {showHidden ? t`Hide hidden games` : t`Show hidden games`}
          </Link>
        </Box>
      ) : null}
      {!games.some((g) => g.available && g.installed) && (
        <Box sx={{ px: 2, py: 0.75, display: 'flex', alignItems: 'center', gap: 1 }}>
          <Typography sx={{ fontSize: 14, color: 'text.secondary' }}>
            {t`No supported game was found in your launchers.`}
          </Typography>
          <Link component="button" onClick={() => openSettings('launchers')} sx={{ fontSize: 14 }}>
            {t`Check launchers…`}
          </Link>
        </Box>
      )}
    </>
  )
})
