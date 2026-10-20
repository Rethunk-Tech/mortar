import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { useCurrentGame } from '../../nav/currentGame.ts'
import { space } from '../../theme/density.ts'
import { PrefByKey } from '../PrefRow.tsx'
import { pickFolderSetting } from '../persist.ts'
import { useSettings } from '../store.ts'

function GameSync() {
  const { t } = useLingui()
  const game = useCurrentGame()
  const own = useSettings((s) => s.games?.[game]?.syncMode === 'own')
  const folder = useSettings((s) => s.games?.[game]?.syncOwnFolder ?? '')
  return (
    <PrefByKey
      prefKey="syncMode"
      game={game}
      extra={
        own ? (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <Typography noWrap={true} title={folder} sx={{ maxWidth: 260 }}>
              {folder || t`No folder chosen`}
            </Typography>
            <Button
              variant="outlined"
              onClick={() =>
                pickFolderSetting(
                  t`Sync folder for this game`,
                  'syncOwnFolder',
                  game,
                  t`Could not save that setting`,
                )
              }
            >
              {t`Choose…`}
            </Button>
          </Box>
        ) : null
      }
    />
  )
}

export { GameSync }
